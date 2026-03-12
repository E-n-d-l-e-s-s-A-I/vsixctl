package usecases

import (
	"context"

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

	// Update Обновляет расширения
	Update(ctx context.Context, ids []domain.ExtensionID, opts UpdateOpts) (UpdateReport, error)
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
