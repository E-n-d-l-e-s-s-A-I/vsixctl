package vscode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type VSCodeStorage struct {
	extensionsPath string
}

func NewVSCodeStorage(extensionsPath string) *VSCodeStorage {
	return &VSCodeStorage{extensionsPath}
}

func (storage *VSCodeStorage) List(ctx context.Context) ([]domain.Extension, error) {
	dirEntries, err := os.ReadDir(storage.extensionsPath)
	if err != nil {
		return nil, fmt.Errorf("list extensions: %w", err)
	}

	result := make([]domain.Extension, 0)

	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}

		extension, err := ParseExtensionDir(filepath.Join(storage.extensionsPath, entry.Name()))
		if err != nil {
			// TODO добавить warning о битой директории с расширением
			continue
		}
		result = append(result, extension)
	}

	return result, nil
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

// Извлекает информацию о расширении из его директории
// Для этого парсит файл package.json
func ParseExtensionDir(dirPath string) (domain.Extension, error) {
	fileContent, err := os.ReadFile(filepath.Join(dirPath, "package.json"))
	if err != nil {
		return domain.Extension{}, fmt.Errorf("parse extension dir: %w", err)
	}

	var pkg packageJSON
	err = json.Unmarshal(fileContent, &pkg)
	if err != nil {
		return domain.Extension{}, fmt.Errorf("parse extension dir: %w", err)
	}

	version, err := domain.ParseVersion(pkg.Version)
	if err != nil {
		return domain.Extension{}, fmt.Errorf("parse extension dir: %w", err)
	}
	return domain.Extension{
		ID: domain.ExtensionID{
			Name:      pkg.Name,
			Publisher: pkg.Publisher,
		},
		Description: pkg.Description,
		Version:     version,
		Platform:    pkg.Metadata.TargetPlatform,
	}, nil
}
