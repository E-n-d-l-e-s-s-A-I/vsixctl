package domain

import "context"

// Storage - абстракция над файловой системой расширений VS Code
type Storage interface {
	// List возвращает все установленные расширения
	List(ctx context.Context) ([]Extension, error)

	// Install устанавливает расширение из .vsix
	Install(ctx context.Context, id ExtensionID, version Version, platform Platform, vsix []byte) error

	// Remove удаляет расширение
	Remove(ctx context.Context, id ExtensionID) error

	// Update обновляет расширение
	Update(ctx context.Context, id ExtensionID, version Version, platform Platform, vsix []byte) error

	// IsInstalled проверяет наличие расширения
	IsInstalled(ctx context.Context, id ExtensionID) (bool, error)

	// InstalledVersion возвращает версию установленного расширения
	InstalledVersion(ctx context.Context, id ExtensionID) (Version, error)
}
