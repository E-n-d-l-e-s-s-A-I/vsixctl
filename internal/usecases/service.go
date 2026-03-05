package usecases

import (
	"context"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type OnProgressFactory func(string) (domain.ProgressFunc, func())

type UseCase interface {
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)
	Install(ctx context.Context, ids []domain.ExtensionID, onProgressFactory OnProgressFactory) []InstallResult
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

func (s *UseCaseService) Install(ctx context.Context, ids []domain.ExtensionID, onProgressFactory OnProgressFactory) []InstallResult {
	results := make([]InstallResult, len(ids))

	// wg чтобы дождаться выполнения всех горутин
	var wg sync.WaitGroup

	// sem чтобы ограничить параллелизм
	sem := make(chan struct{}, s.parallelism)

	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				onProgress, exitFunc := onProgressFactory(id.String())
				defer exitFunc()
				res := s.installExtension(ctx, id, onProgress)
				// Mutex не нужен, т.к. каждая горутина работает со своей областью памяти
				results[i] = res
			case <-ctx.Done():
				// контекст отменён, выходим
				results[i] = InstallResult{id, ctx.Err()}
				return
			}
		}()
	}
	wg.Wait()
	return results
}

func (s *UseCaseService) Update(ctx context.Context) error {
	return nil
}

func (s *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return s.storage.List(ctx)
}

func (s *UseCaseService) installExtension(ctx context.Context, id domain.ExtensionID, onProgress domain.ProgressFunc) InstallResult {
	latestVer, err := s.registry.GetLatestVersion(ctx, id)
	if err != nil {
		return InstallResult{id, err}
	}
	data, err := s.registry.Download(ctx, latestVer, onProgress)
	if err != nil {
		return InstallResult{id, err}
	}

	err = s.storage.Install(ctx, id, latestVer.Version, data)

	return InstallResult{id, err}
}
