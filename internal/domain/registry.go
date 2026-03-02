package domain

import (
	"context"
	"io"
)

// ProgressFunc - callback прогресса скачивания
// total может быть -1 если размер неизвестен
type ProgressFunc func(downloaded, total int64)

// Registry - абстракция маркетплейса (VS Code Marketplace, Open VSX, etc.)
type Registry interface {
	// Search ищет расширения по запросу
	Search(ctx context.Context, query string, count int) ([]Extension, error)

	// GetLatestVersion возвращает последнюю версию расширения
	GetLatestVersion(ctx context.Context, id ExtensionID) (VersionInfo, error)

	// Download скачивает .vsix пакет, вызывая onProgress по мере скачивания
	Download(ctx context.Context, version VersionInfo, onProgress ProgressFunc) (io.ReadCloser, error)
}
