package usecases

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Search поиск расширений
func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
}
