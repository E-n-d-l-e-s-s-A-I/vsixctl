package testutil

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// MockRegistry — мок domain.Registry с функциональными полями.
// Незаданные методы паникуют, чтобы тест явно падал при неожиданном вызове.
type MockRegistry struct {
	SearchFunc          func(ctx context.Context, query string, count int) ([]domain.Extension, error)
	GetDownloadInfoFunc func(ctx context.Context, id domain.ExtensionID) (domain.Extension, domain.DownloadInfo, error)
	DownloadFunc        func(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) ([]byte, error)
}

func (m *MockRegistry) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	if m.SearchFunc == nil {
		panic("MockRegistry.SearchFunc not set")
	}
	return m.SearchFunc(ctx, query, count)
}

func (m *MockRegistry) GetDownloadInfo(ctx context.Context, id domain.ExtensionID) (domain.Extension, domain.DownloadInfo, error) {
	if m.GetDownloadInfoFunc == nil {
		panic("MockRegistry.GetDownloadInfoFunc not set")
	}
	return m.GetDownloadInfoFunc(ctx, id)
}

func (m *MockRegistry) Download(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) ([]byte, error) {
	if m.DownloadFunc == nil {
		panic("MockRegistry.DownloadFunc not set")
	}
	return m.DownloadFunc(ctx, info, onProgress)
}
