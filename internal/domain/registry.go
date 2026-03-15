package domain

import "context"

// Registry - абстракция маркетплейса (VS Code Marketplace, Open VSX, etc.)
type Registry interface {
	// Search ищет расширения по запросу
	Search(ctx context.Context, query string, count int) ([]Extension, error)

	// GetDownloadInfo возвращает расширение и мета-информацию для его установки
	GetDownloadInfo(ctx context.Context, id ExtensionID, version *Version) (Extension, DownloadInfo, error)

	// Download скачивает .vsix пакет, вызывая onProgress по мере скачивания
	Download(ctx context.Context, info DownloadInfo, onProgress ProgressFunc) ([]byte, error)

	// Получает версии расширения
	GetVersions(ctx context.Context, id ExtensionID, limit int) ([]VersionInfo, error)
}
