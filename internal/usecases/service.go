package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// TODO можно сразу же фиксировать и size
type OnProgressFactory func(string) (domain.ProgressFunc, func())

type UseCase interface {
	// List возвращает установленные расширения
	List(ctx context.Context) ([]domain.Extension, error)

	// Install устанавливает расширения
	Install(ctx context.Context, ids []domain.ExtensionID, opts InstallOpts) (InstallReport, error)

	// Search поиск расширений
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)

	// RemoveResolve возвращает удаляемые расширения, и расширения которые не установлены
	RemoveResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.Extension, notInstalled []domain.ExtensionID, err error)

	// Remove удаляет расширения
	Remove(ctx context.Context, ids []domain.ExtensionID) []domain.ExtensionResult

	// UpdateResolve возвращает расширения, которые будут удалены(устаревшие версии) - resolved.prev, расширения, которые будут установлены - resolved.new(новые версии)
	// И расширения, которые были запрошены, но не установлены - notInstalled
	// Если в качестве ids передать пустой список, то будут проверенны все установленные расширения
	UpdateResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.UpdateInfo, notInstalled []domain.ExtensionID, err error)

	// Обновляет расширения, атомарно заменяя старые версии на новые
	Update(ctx context.Context, resolved []domain.UpdateInfo, onProgressFactory OnProgressFactory) ([]domain.ExtensionResult, error)
}

type UseCaseService struct {
	registry    domain.Registry
	storage     domain.Storage
	onStatus    func(string)
	parallelism int // Кол-во параллельных загрузок
}

func NewUseCaseService(registry domain.Registry, storage domain.Storage, onStatus func(string), parallelism int) *UseCaseService {
	if onStatus == nil {
		onStatus = func(string) {}
	}
	return &UseCaseService{
		registry:    registry,
		storage:     storage,
		onStatus:    onStatus,
		parallelism: parallelism,
	}
}

// Search поиск расширений
func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
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
			updateInfo, err := domain.NewUpdateInfo(ext, latest)
			if err != nil {
				return nil, nil, fmt.Errorf("update resolve: %w", err)
			}
			resolved = append(resolved, updateInfo)
		}
	}
	return resolved, notInstalled, nil
}

func (s *UseCaseService) Update(ctx context.Context, resolved []domain.UpdateInfo, onProgressFactory OnProgressFactory) ([]domain.ExtensionResult, error) {
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

	return results, nil
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
