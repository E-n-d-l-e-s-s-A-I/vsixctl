package vscode

import (
	"context"
	"io"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type VSCodeStorage struct {
	extensionsPath string
}

func NewVSCodeStorage(extensionsPath string) *VSCodeStorage {
	return &VSCodeStorage{extensionsPath}
}

func (storage *VSCodeStorage) List(ctx context.Context) ([]domain.Extension, error) {
	return nil, nil
}

func (storage *VSCodeStorage) Install(ctx context.Context, id domain.ExtensionID, version domain.Version, vsix io.Reader) error {
	return nil
}

func (storage *VSCodeStorage) Remove(ctx context.Context, id domain.ExtensionID) error {
	return nil
}

func (storage *VSCodeStorage) IsInstalled(ctx context.Context, id domain.ExtensionID) (bool, error) {
	return false, nil
}

func (storage *VSCodeStorage) InstalledVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	return domain.Version{}, nil
}
