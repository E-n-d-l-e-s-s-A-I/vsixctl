package domain

import (
	"context"
	"io"
)

// ProgressFunc — callback прогресса скачивания
// total может быть -1 если размер неизвестен
type ProgressFunc func(downloaded, total int64)

// Registry — абстракция маркетплейса (VS Code Marketplace, Open VSX, etc.)
type Registry interface {
	// Search ищет расширения по запросу
	Search(ctx context.Context, query string) ([]SearchResult, error)

	// GetLatestVersion возвращает последнюю версию расширения
	GetLatestVersion(ctx context.Context, id ExtensionID) (Version, error)

	// Download скачивает .vsix пакет, вызывая onProgress по мере скачивания
	Download(ctx context.Context, id ExtensionID, version Version, onProgress ProgressFunc) (io.ReadCloser, error)
}
