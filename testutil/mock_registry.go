package testutil

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// MockRegistry - мок domain.Registry с функциональными полями.
// Незаданные методы паникуют, чтобы тест явно падал при неожиданном вызове.
type MockRegistry struct {
	SearchFunc          func(ctx context.Context, query domain.SearchQuery) ([]domain.Extension, error)
	GetDownloadInfoFunc func(ctx context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error)
	DownloadFunc        func(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) ([]byte, error)
	GetVersionsFunc     func(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error)
}

func (m *MockRegistry) Search(ctx context.Context, query domain.SearchQuery) ([]domain.Extension, error) {
	if m.SearchFunc == nil {
		panic("MockRegistry.SearchFunc not set")
	}
	return m.SearchFunc(ctx, query)
}

func (m *MockRegistry) GetDownloadInfo(ctx context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
	if m.GetDownloadInfoFunc == nil {
		panic("MockRegistry.GetDownloadInfoFunc not set")
	}
	return m.GetDownloadInfoFunc(ctx, id, version)
}

func (m *MockRegistry) Download(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) ([]byte, error) {
	if m.DownloadFunc == nil {
		panic("MockRegistry.DownloadFunc not set")
	}
	return m.DownloadFunc(ctx, info, onProgress)
}

func (m *MockRegistry) GetVersions(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
	if m.GetVersionsFunc == nil {
		panic("MockRegistry.GetVersionsFunc not set")
	}
	return m.GetVersionsFunc(ctx, id, limit)
}
