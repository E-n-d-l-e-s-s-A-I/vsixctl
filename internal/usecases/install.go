package usecases

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type InstallOpts struct {
	Confirm           func(requested []domain.ExtensionID, toInstall []domain.DownloadInfo, toReinstall []domain.ReinstallInfo) bool
	OnProgressFactory OnProgressFactory
	Force             bool
}

type InstallReport struct {
	Results []domain.ExtensionResult
}

// Install установка расширений
func (s *UseCaseService) Install(ctx context.Context, targets []domain.InstallTarget, opts InstallOpts) (InstallReport, error) {
	s.onStatus("resolving dependencies...")

	requestedIDs := make([]domain.ExtensionID, len(targets))
	for i, t := range targets {
		requestedIDs[i] = t.ID
	}

	resolved, alreadyInstalled, reinstall, err := s.installResolve(ctx, targets, requestedIDs, opts.Force)

	if err != nil {
		return InstallReport{}, fmt.Errorf("install: %w", err)
	}

	// Формируем результаты для уже установленных расширений
	var results []domain.ExtensionResult
	for _, id := range alreadyInstalled {
		results = append(results, domain.ExtensionResult{ID: id, Err: domain.ErrAlreadyInstalled})
	}

	if len(resolved) == 0 && len(reinstall) == 0 {
		return InstallReport{Results: results}, nil
	}

	if ok := opts.Confirm(requestedIDs, resolved, reinstall); !ok {
		return InstallReport{Results: results}, nil
	}

	// Объединяем новые установки и переустановки для скачивания
	toDownload := resolved
	for _, ri := range reinstall {
		toDownload = append(toDownload, ri.New)
	}

	installResults := s.downloadAndInstall(ctx, toDownload, opts.OnProgressFactory)
	results = append(results, installResults...)
	return InstallReport{Results: results}, nil
}

// installResolve резолв всех расширений и их зависимостей
func (s *UseCaseService) installResolve(ctx context.Context, targets []domain.InstallTarget, requestedIDs []domain.ExtensionID, force bool) (resolved []domain.DownloadInfo, alreadyInstalled []domain.ExtensionID, reinstall []domain.ReinstallInfo, err error) {
	resolved, err = s.installResolveAll(ctx, targets)
	if err != nil {
		return nil, nil, nil, err
	}

	// Фильтрация уже установленных
	installed, resolved, reinstall, err := s.filterInstalled(ctx, resolved, requestedIDs, force)
	if err != nil {
		return nil, nil, nil, err
	}

	return resolved, installed, reinstall, nil
}

// downloadAndInstall асинхронно скачивает и устанавливает расширения
func (s *UseCaseService) downloadAndInstall(ctx context.Context, extensions []domain.DownloadInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, s.parallelism)
		results = make([]domain.ExtensionResult, len(extensions))
	)
	for i, info := range extensions {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(info.ID.String(), info.Size)
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				results[i] = s.installExtension(ctx, info, onProgress)
				if results[i].Err == nil {
					onProgress(info.Size)
				}
			case <-ctx.Done():
				results[i] = domain.ExtensionResult{ID: info.ID, Err: ctx.Err()}
				return
			}
		}()
	}
	wg.Wait()
	return results
}

// installResolveAll резолвит зависимости всех переданных устанавливаемых расширений
func (s *UseCaseService) installResolveAll(ctx context.Context, ids []domain.InstallTarget) ([]domain.DownloadInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		visited    sync.Map
		mu         sync.Mutex
		wg         sync.WaitGroup
		sem        = make(chan struct{}, s.parallelism)
		once       sync.Once
		resolveErr error
		resolved   []domain.DownloadInfo
	)

	var resolve func(domain.InstallTarget)
	resolve = func(target domain.InstallTarget) {
		defer wg.Done()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			ext, downloadInfo, err := s.registry.GetDownloadInfo(ctx, target.ID, target.Version)
			if err != nil {
				once.Do(func() {
					resolveErr = fmt.Errorf("%s: %w", target.ID, err)
					cancel()
				})
				return
			}

			mu.Lock()
			resolved = append(resolved, downloadInfo)
			mu.Unlock()

			for _, dep := range ext.ExtensionPack {
				if dep.Publisher == domain.BuiltInPublisher {
					continue
				}
				if _, loaded := visited.LoadOrStore(dep, struct{}{}); !loaded {
					wg.Add(1)
					go resolve(domain.InstallTarget{ID: dep})
				}
			}
			for _, dep := range ext.Dependencies {
				if dep.Publisher == domain.BuiltInPublisher {
					continue
				}
				if _, loaded := visited.LoadOrStore(dep, struct{}{}); !loaded {
					wg.Add(1)
					go resolve(domain.InstallTarget{ID: dep})
				}
			}

		case <-ctx.Done():
			return
		}
	}

	for _, target := range ids {
		if _, loaded := visited.LoadOrStore(target.ID, struct{}{}); !loaded {
			wg.Add(1)
			go resolve(target)
		}
	}

	wg.Wait()
	return resolved, resolveErr
}

// filterInstalled отделяет уже установленные расширения из resolved.
// Возвращает ID установленных (только тех, что пользователь явно запросил), отфильтрованный список для скачивания
// и список переустановок (при force=true).
func (s *UseCaseService) filterInstalled(ctx context.Context, resolved []domain.DownloadInfo, requestedIDs []domain.ExtensionID, force bool) (installed []domain.ExtensionID, filteredResolved []domain.DownloadInfo, reinstall []domain.ReinstallInfo, err error) {
	installedExtensions, err := s.storage.List(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	installedMap := make(map[domain.ExtensionID]domain.Extension, len(installedExtensions))
	for _, ext := range installedExtensions {
		installedMap[ext.ID] = ext
	}

	for _, ext := range resolved {
		if prev, ok := installedMap[ext.ID]; ok {
			if slices.Contains(requestedIDs, ext.ID) {
				if force {
					reinstall = append(reinstall, domain.ReinstallInfo{Prev: prev, New: ext})
				} else {
					installed = append(installed, ext.ID)
				}
			}
		} else {
			filteredResolved = append(filteredResolved, ext)
		}
	}
	return installed, filteredResolved, reinstall, nil
}

// installExtension скачивает и устанавливает одно расширение
func (s *UseCaseService) installExtension(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) domain.ExtensionResult {
	data, err := s.registry.Download(ctx, info, onProgress)
	if err != nil {
		return domain.ExtensionResult{ID: info.ID, Err: err}
	}

	err = s.storage.Install(ctx, domain.InstallParams{
		ID:       info.ID,
		Version:  info.Version,
		Platform: info.Platform,
		Meta:     info.Meta,
		Data:     data,
	})

	return domain.ExtensionResult{ID: info.ID, Err: err}
}
