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

	// Remove удаляет расширения
	Remove(ctx context.Context, ids []domain.ExtensionID, opts RemoveOpts) (RemoveReport, error)

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
