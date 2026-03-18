package usecases

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Versions возвращает список версий расширения
func (s *UseCaseService) Versions(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
	return s.registry.GetVersions(ctx, id, limit)
}
