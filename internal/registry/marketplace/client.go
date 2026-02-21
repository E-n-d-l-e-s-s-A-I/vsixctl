package marketplace

import (
	"context"
	"io"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type MarketplaceRegistry struct {
	URL string
}

func (marketplace *MarketplaceRegistry) Search(ctx context.Context, query string) ([]domain.SearchResult, error) {
	a := make([]domain.SearchResult, 0)
	return a, nil
}

func (marketplace *MarketplaceRegistry) GetLatestVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	return domain.Version{}, nil
}

func (marketplace *MarketplaceRegistry) Download(ctx context.Context, id domain.ExtensionID, version domain.Version, onProgress domain.ProgressFunc) (io.ReadCloser, error) {
	return nil, nil
}
