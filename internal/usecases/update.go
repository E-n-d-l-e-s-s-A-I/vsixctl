package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type UpdateOpts struct {
	Confirm           func(toUpdate []domain.UpdateInfo) bool
	OnProgressFactory OnProgressFactory
}

type UpdateReport struct {
	Results []domain.ExtensionResult
}

// Update обновляет расширения
func (s *UseCaseService) Update(ctx context.Context, ids []domain.ExtensionID, opts UpdateOpts) (UpdateReport, error) {
	s.onStatus("search for updates...")
	resolved, notInstalled, err := s.updateResolve(ctx, ids)
	if err != nil {
		return UpdateReport{}, fmt.Errorf("update: %w", err)
	}

	// Формируем результаты для неустановленных расширений
	var results []domain.ExtensionResult
	for _, id := range notInstalled {
		results = append(results, domain.ExtensionResult{ID: id, Err: domain.ErrNotInstalled})
	}
	if len(resolved) == 0 {
		return UpdateReport{Results: results}, nil
	}

	if ok := opts.Confirm(resolved); !ok {
		return UpdateReport{Results: results}, nil
	}

	updateResults := s.update(ctx, resolved, opts.OnProgressFactory)
	results = append(results, updateResults...)

	return UpdateReport{Results: results}, nil
}

func (s *UseCaseService) updateResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.UpdateInfo, notInstalled []domain.ExtensionID, err error) {
	installed, err := s.storage.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("update resolve: %w", err)
	}
	idToInstalled := make(map[domain.ExtensionID]domain.Extension, len(installed))
	for _, ext := range installed {
		idToInstalled[ext.ID] = ext
	}

	requested := make([]domain.Extension, 0)
	if len(ids) != 0 {
		for _, id := range ids {
			installedExt, ok := idToInstalled[id]
			if !ok {
				notInstalled = append(notInstalled, id)
				continue
			}
			requested = append(requested, installedExt)
		}
	} else {
		// Расширения не переданы => запрашиваются все
		requested = installed
	}

	// Получаем последние версии расширений
	// TODO меня смущает что мы обращаемся к installResolveAllinstallResolveAll
	// И тянем лишние данные о зависимостях
	requestedIds := make([]domain.ExtensionID, len(requested))
	for i, ext := range requested {
		requestedIds[i] = ext.ID
	}
	latestVersions, err := s.installResolveAll(ctx, requestedIds)
	if err != nil {
		return nil, nil, fmt.Errorf("update resolve: %w", err)
	}
	idToLatestVer := make(map[domain.ExtensionID]domain.DownloadInfo)
	for _, ver := range latestVersions {
		idToLatestVer[ver.ID] = ver
	}

	// Оставляем только те расширения, для которых вышла новая версия
	for _, ext := range requested {
		latest, ok := idToLatestVer[ext.ID]
		if !ok {
			return nil, nil, fmt.Errorf("update resolve: latest version not found by unknown reason")
		}
		if latest.Version.NewerThan(ext.Version) {
			updateInfo, err := domain.NewUpdateInfo(ext, latest)
			if err != nil {
				return nil, nil, fmt.Errorf("update resolve: %w", err)
			}
			resolved = append(resolved, updateInfo)
		}
	}
	return resolved, notInstalled, nil
}

func (s *UseCaseService) update(ctx context.Context, resolved []domain.UpdateInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.parallelism)
	results := make([]domain.ExtensionResult, len(resolved))

	for i, ext := range resolved {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(ext.New.ID.String())
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				vsix, err := s.registry.Download(ctx, ext.New, onProgress)
				if err != nil {
					results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: err}
					return
				}
				err = s.storage.Update(ctx, ext.New.ID, ext.New.Version, ext.New.Platform, vsix)
				results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: err}
			case <-ctx.Done():
				results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: ctx.Err()}
			}
		}()
	}
	wg.Wait()

	return results
}
