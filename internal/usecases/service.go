package usecases

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type OnProgressFactory func(string) (domain.ProgressFunc, func())

type UseCase interface {
	// List возвращает установленные расширения
	List(ctx context.Context) ([]domain.Extension, error)

	// Search поиск расширений
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)

	// InstallResolve возвращает мета-данные для скачивания, и расширения которые уже установлены
	InstallResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.DownloadInfo, alreadyInstalled []domain.ExtensionID, err error)

	// Install устанавливает расширения
	Install(ctx context.Context, extensions []domain.DownloadInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult

	// RemoveResolve возвращает удаляемые расширения, и расширения которые не установлены
	RemoveResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.Extension, notInstalled []domain.ExtensionID, err error)

	// Remove удаляет расширения
	Remove(ctx context.Context, ids []domain.ExtensionID) []domain.ExtensionResult

	// UpdateResolve возвращает расширения, которые будут удалены(устаревшие версии) - resolved.prev, расширения, которые будут установлены - resolved.new(новые версии)
	// И расширения, которые были запрошены, но не установлены - notInstalled
	// Если в качестве ids передать пустой список, то будут проверенны все установленные расширения
	UpdateResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.UpdateInfo, notInstalled []domain.ExtensionID, err error)

	// Обновляет расширения, атомарно заменяя старые версии на новые
	Update(ctx context.Context, resolved []domain.UpdateInfo) ([]domain.ExtensionResult, error)
}

type UseCaseService struct {
	registry    domain.Registry
	storage     domain.Storage
	parallelism int // Кол-во параллельных загрузок
}

func NewUseCaseService(registry domain.Registry, storage domain.Storage, parallelism int) *UseCaseService {
	return &UseCaseService{
		registry:    registry,
		storage:     storage,
		parallelism: parallelism,
	}
}

// Search поиск расширений
func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
}

// Resolve резолв всех расширений и их зависимостей
func (s *UseCaseService) InstallResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.DownloadInfo, alreadyInstalled []domain.ExtensionID, err error) {
	resolved, err = s.installResolveAll(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	// Фильтрация уже установленных
	installed, resolved, err := s.filterInstalled(ctx, resolved, ids)
	if err != nil {
		return nil, nil, err
	}

	return resolved, installed, nil
}

func (s *UseCaseService) Install(ctx context.Context, extensions []domain.DownloadInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	return s.downloadAndInstall(ctx, extensions, onProgressFactory)
}

// Remove удаление расширений
func (s *UseCaseService) Remove(ctx context.Context, ids []domain.ExtensionID) []domain.ExtensionResult {
	results := make([]domain.ExtensionResult, len(ids))
	for i, id := range ids {
		err := s.storage.Remove(ctx, id)
		results[i] = domain.ExtensionResult{ID: id, Err: err}
	}
	return results
}

func (s *UseCaseService) UpdateResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.UpdateInfo, notInstalled []domain.ExtensionID, err error) {
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
			resolved = append(resolved, domain.UpdateInfo{
				Prev: ext,
				New:  latest,
			})
		}
	}
	return resolved, notInstalled, nil
}

// TODO реализовать
func (s *UseCaseService) Update(ctx context.Context, resolved []domain.UpdateInfo) ([]domain.ExtensionResult, error) {
	return nil, nil
}

// filterInstalled отделяет уже установленные расширения из resolved.
// Возвращает ID установленных (только тех, что пользователь явно запросил) и отфильтрованный список для скачивания.
func (s *UseCaseService) filterInstalled(ctx context.Context, resolved []domain.DownloadInfo, requestedIDs []domain.ExtensionID) (installed []domain.ExtensionID, filteredResolved []domain.DownloadInfo, err error) {
	installedExtensions, err := s.storage.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	installedMap := make(map[domain.ExtensionID]struct{}, len(installedExtensions))
	for _, ext := range installedExtensions {
		installedMap[ext.ID] = struct{}{}
	}

	for _, ext := range resolved {
		if _, ok := installedMap[ext.ID]; ok {
			if slices.Contains(requestedIDs, ext.ID) {
				installed = append(installed, ext.ID)
			}
		} else {
			filteredResolved = append(filteredResolved, ext)
		}
	}
	return installed, filteredResolved, nil
}

// downloadAndInstall асинхронно скачивает и устанавливает расширения
func (s *UseCaseService) downloadAndInstall(ctx context.Context, extensions []domain.DownloadInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		sem     = make(chan struct{}, s.parallelism)
		results []domain.ExtensionResult
	)
	for _, info := range extensions {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(info.ID.String())
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				res := s.installExtension(ctx, info, onProgress)
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			case <-ctx.Done():
				mu.Lock()
				results = append(results, domain.ExtensionResult{ID: info.ID, Err: ctx.Err()})
				mu.Unlock()
				return
			}
		}()
	}
	wg.Wait()
	return results
}

func (s *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return s.storage.List(ctx)
}

// installExtension скачивает и устанавливает одно расширение
func (s *UseCaseService) installExtension(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) domain.ExtensionResult {
	data, err := s.registry.Download(ctx, info, onProgress)
	if err != nil {
		return domain.ExtensionResult{ID: info.ID, Err: err}
	}

	err = s.storage.Install(ctx, info.ID, info.Version, info.Platform, data)

	return domain.ExtensionResult{ID: info.ID, Err: err}
}

// installResolveAll резолвит зависимости всех переданных устанавливаемых расширений
func (s *UseCaseService) installResolveAll(ctx context.Context, ids []domain.ExtensionID) ([]domain.DownloadInfo, error) {
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

	var resolve func(domain.ExtensionID)
	resolve = func(id domain.ExtensionID) {
		defer wg.Done()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			ext, downloadInfo, err := s.registry.GetDownloadInfo(ctx, id)
			if err != nil {
				once.Do(func() {
					resolveErr = err
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
					go resolve(dep)
				}
			}
			for _, dep := range ext.Dependencies {
				if dep.Publisher == domain.BuiltInPublisher {
					continue
				}
				if _, loaded := visited.LoadOrStore(dep, struct{}{}); !loaded {
					wg.Add(1)
					go resolve(dep)
				}
			}

		case <-ctx.Done():
			return
		}
	}

	for _, id := range ids {
		if _, loaded := visited.LoadOrStore(id, struct{}{}); !loaded {
			wg.Add(1)
			go resolve(id)
		}
	}

	wg.Wait()
	return resolved, resolveErr
}

// RemoveResolve резолвит все удаляемые расширения
// Добавляет к самим расширениям их пакетные расширения, а так же фильтрует не установленные
func (s *UseCaseService) RemoveResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.Extension, notInstalled []domain.ExtensionID, err error) {
	installed, err := s.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve remove: %w", err)
	}
	installedMap := make(map[domain.ExtensionID]domain.Extension, len(installed))
	for _, ext := range installed {
		installedMap[ext.ID] = ext
	}

	for _, id := range ids {
		pack := resolvePack(id, installedMap)
		if len(pack) == 0 {
			notInstalled = append(notInstalled, id)
			continue
		}
		resolved = append(resolved, pack...)
	}
	resolved = uniqExtensions(resolved)

	return resolved, notInstalled, nil
}

// resolvePack рекурсивно возвращает все расширения из пакета, которые установлены
func resolvePack(id domain.ExtensionID, installed map[domain.ExtensionID]domain.Extension) []domain.Extension {
	var result []domain.Extension
	seen := make(map[domain.ExtensionID]struct{})

	var resolve func(id domain.ExtensionID)
	resolve = func(id domain.ExtensionID) {
		if _, ok := seen[id]; ok {
			return
		}

		seen[id] = struct{}{}
		ext, ok := installed[id]
		if !ok {
			return
		}
		result = append(result, ext)
		for _, packExt := range ext.ExtensionPack {
			resolve(packExt)
		}
	}
	resolve(id)

	return result
}

// uniqExtensions дедуплицирует список расширений
func uniqExtensions(extensions []domain.Extension) []domain.Extension {
	var result []domain.Extension
	seen := make(map[domain.ExtensionID]struct{})

	for _, ext := range extensions {
		if _, ok := seen[ext.ID]; ok {
			continue
		}
		result = append(result, ext)
		seen[ext.ID] = struct{}{}
	}

	return result
}
