package usecases

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// List выводит список расширений
func (s *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return s.storage.List(ctx)
}
