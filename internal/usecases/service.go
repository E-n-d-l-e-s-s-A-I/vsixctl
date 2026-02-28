package usecases

import (
	"context"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type UseCase interface {
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)
	Install(ctx context.Context, ids []domain.ExtensionID) []InstallResult
	Update(ctx context.Context) error
	List(ctx context.Context) ([]domain.Extension, error)
}

type UseCaseService struct {
	registry domain.Registry
	storage  domain.Storage
}

func NewUserCaseService(registry domain.Registry, storage domain.Storage) *UseCaseService {
	return &UseCaseService{
		registry: registry,
		storage:  storage,
	}
}

func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
}

// TODO протестировать
func (s *UseCaseService) Install(ctx context.Context, ids []domain.ExtensionID) []InstallResult {
	results := make([]InstallResult, len(ids))

	// wg чтобы дождаться выполнения всех горутин
	var wg sync.WaitGroup

	// sem чтобы ограничить параллелизм
	// TODO временный параллелизма в 2
	sem := make(chan struct{}, 2)

	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				res := s.installExtension(ctx, id)
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

func (s *UseCaseService) installExtension(ctx context.Context, id domain.ExtensionID) InstallResult {
	latestVer, err := s.registry.GetLatestVersion(ctx, id)
	if err != nil {
		return InstallResult{id, err}
	}
	reader, err := s.registry.Download(ctx, id, latestVer, func(downloaded, total int64) {})
	if err != nil {
		return InstallResult{id, err}
	}
	defer reader.Close()

	err = s.storage.Install(ctx, id, latestVer, reader)

	return InstallResult{id, err}
}
