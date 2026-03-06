package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type OnProgressFactory func(string) (domain.ProgressFunc, func())

type UseCase interface {
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)
	Install(ctx context.Context, ids []domain.ExtensionID, onProgressFactory OnProgressFactory) ([]domain.InstallResult, error)
	Update(ctx context.Context) error
	List(ctx context.Context) ([]domain.Extension, error)
}

type UseCaseService struct {
	registry    domain.Registry
	storage     domain.Storage
	parallelism int // Кол-во параллельных загрузок
}

func NewUserCaseService(registry domain.Registry, storage domain.Storage, parallelism int) *UseCaseService {
	return &UseCaseService{
		registry:    registry,
		storage:     storage,
		parallelism: parallelism,
	}
}

func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
}

func (s *UseCaseService) Install(ctx context.Context, ids []domain.ExtensionID, onProgressFactory OnProgressFactory) ([]domain.InstallResult, error) {
	results := make([]domain.InstallResult, len(ids))
	installedExtensions, err := s.storage.List(ctx)
	if err != nil {
		return nil, err
	}
	installedExtensionsMap := make(map[domain.ExtensionID]domain.Extension, len(installedExtensions))
	for _, ext := range installedExtensions {
		installedExtensionsMap[ext.ID] = ext
	}

	// wg чтобы дождаться выполнения всех горутин
	var wg sync.WaitGroup

	// sem чтобы ограничить параллелизм
	sem := make(chan struct{}, s.parallelism)

	for i, id := range ids {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(id.String())
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				res := s.installExtension(ctx, id, installedExtensionsMap, onProgress)
				// Mutex не нужен, т.к. каждая горутина работает со своей областью памяти
				results[i] = res
			case <-ctx.Done():
				// контекст отменён, выходим
				results[i] = domain.InstallResult{ID: id, Err: ctx.Err()}
				return
			}
		}()
	}
	wg.Wait()
	return results, nil
}

func (s *UseCaseService) Update(ctx context.Context) error {
	return nil
}

func (s *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return s.storage.List(ctx)
}

func (s *UseCaseService) installExtension(ctx context.Context, id domain.ExtensionID, installedExtensions map[domain.ExtensionID]domain.Extension, onProgress domain.ProgressFunc) domain.InstallResult {
	if _, ok := installedExtensions[id]; ok {
		return domain.InstallResult{ID: id, Err: fmt.Errorf("install extension: %w", domain.ErrAlreadyInstalled)}
	}

	latestVer, err := s.registry.GetLatestVersion(ctx, id)
	if err != nil {
		return domain.InstallResult{ID: id, Err: err}
	}
	data, err := s.registry.Download(ctx, latestVer, onProgress)
	if err != nil {
		return domain.InstallResult{ID: id, Err: err}
	}

	err = s.storage.Install(ctx, id, latestVer.Version, data)

	return domain.InstallResult{ID: id, Err: err}
}
