package testutil

import (
	"context"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// MockStorage - мок domain.Storage с функциональными полями.
// Незаданные методы паникуют, чтобы тест явно падал при неожиданном вызове.
type MockStorage struct {
	ListFunc             func(ctx context.Context) ([]domain.Extension, error)
	InstallFunc          func(ctx context.Context, params domain.InstallParams) error
	RemoveFunc           func(ctx context.Context, id domain.ExtensionID) error
	UpdateFunc           func(ctx context.Context, params domain.InstallParams) error
	IsInstalledFunc      func(ctx context.Context, id domain.ExtensionID) (bool, error)
	InstalledVersionFunc func(ctx context.Context, id domain.ExtensionID) (domain.Version, error)
}

func (m *MockStorage) List(ctx context.Context) ([]domain.Extension, error) {
	if m.ListFunc == nil {
		panic("MockStorage.ListFunc not set")
	}
	return m.ListFunc(ctx)
}

func (m *MockStorage) Install(ctx context.Context, params domain.InstallParams) error {
	if m.InstallFunc == nil {
		panic("MockStorage.InstallFunc not set")
	}
	return m.InstallFunc(ctx, params)
}

func (m *MockStorage) Remove(ctx context.Context, id domain.ExtensionID) error {
	if m.RemoveFunc == nil {
		panic("MockStorage.RemoveFunc not set")
	}
	return m.RemoveFunc(ctx, id)
}

func (m *MockStorage) Update(ctx context.Context, params domain.InstallParams) error {
	if m.UpdateFunc == nil {
		panic("MockStorage.UpdateFunc not set")
	}
	return m.UpdateFunc(ctx, params)
}

func (m *MockStorage) IsInstalled(ctx context.Context, id domain.ExtensionID) (bool, error) {
	if m.IsInstalledFunc == nil {
		panic("MockStorage.IsInstalledFunc not set")
	}
	return m.IsInstalledFunc(ctx, id)
}

func (m *MockStorage) InstalledVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	if m.InstalledVersionFunc == nil {
		panic("MockStorage.InstalledVersionFunc not set")
	}
	return m.InstalledVersionFunc(ctx, id)
}
