package usecases

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type UseCase interface {
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)
	Install(ctx context.Context, id domain.ExtensionID) error
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

func (service *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return service.registry.Search(ctx, query, count)
}

func (service *UseCaseService) Install(ctx context.Context, id domain.ExtensionID) error {
	return nil
}

func (service *UseCaseService) Update(ctx context.Context) error {
	return nil
}

func (service *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return service.storage.List(ctx)
}
