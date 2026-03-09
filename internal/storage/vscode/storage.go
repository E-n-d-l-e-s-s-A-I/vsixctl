package vscode

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

const registryFileName = "extensions.json"

type VSCodeStorage struct {
	extensionsPath string
	logFunc        domain.LogFunc
	mu             sync.Mutex
}

func NewVSCodeStorage(extensionsPath string, logFunc domain.LogFunc) *VSCodeStorage {
	if logFunc == nil {
		logFunc = func(string) {}
	}
	return &VSCodeStorage{
		extensionsPath: extensionsPath,
		logFunc:        logFunc,
	}
}

func (s *VSCodeStorage) List(ctx context.Context) ([]domain.Extension, error) {
	dirEntries, err := os.ReadDir(s.extensionsPath)
	if err != nil {
		return nil, fmt.Errorf("list extensions: %w", err)
	}

	result := make([]domain.Extension, 0)

	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}

		extDir := filepath.Join(s.extensionsPath, entry.Name())
		extension, err := ParseExtensionDir(extDir)
		if err != nil {
			s.logFunc(fmt.Sprintf("failed to parse extension directory %s: %v", extDir, err))
			continue
		}
		result = append(result, extension)
	}

	return result, nil
}

func (s *VSCodeStorage) Install(ctx context.Context, id domain.ExtensionID, version domain.VersionInfo, vsix []byte) error {
	tmpFile, err := saveToTempFile(vsix)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	info, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	extDirName := fmt.Sprintf("%s.%s-%s", id.Publisher, id.Name, version.Version.String())
	destDir := filepath.Join(s.extensionsPath, extDirName)
	zipReader, err := zip.NewReader(tmpFile, info.Size())
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if err := unpackVsix(zipReader, destDir); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	if err := s.registerExtension(id, version.Version, extDirName); err != nil {
		// Удаляем директорию расширения при ошибки регистрации в реестре
		if err := os.RemoveAll(destDir); err != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, err))
		}
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

func (s *VSCodeStorage) Remove(ctx context.Context, id domain.ExtensionID) error {
	return nil
}

func (s *VSCodeStorage) IsInstalled(ctx context.Context, id domain.ExtensionID) (bool, error) {
	return false, nil
}

func (s *VSCodeStorage) InstalledVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
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

// Сохраняет данные во временный файл
func saveToTempFile(data []byte) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "vsixctl-*.vsix")
	if err != nil {
		return nil, fmt.Errorf("save to temp file: %w", err)
	}

	_, err = tmpFile.Write(data)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("save to temp file: %w", err)
	}

	return tmpFile, nil
}

// Извлекает одну запись из zip-архива в targetPath
func extractZipFile(f *zip.File, targetPath string) error {
	err := os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return fmt.Errorf("extract zip file: %w", err)
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("extract zip file: %w", err)
	}
	defer file.Close()

	reader, err := f.Open()
	if err != nil {
		return fmt.Errorf("extract zip file: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("extract zip file: %w", err)
	}

	return nil
}

// Регистрирует расширение в реестре VS Code (extensions.json)
func (s *VSCodeStorage) registerExtension(id domain.ExtensionID, ver domain.Version, relativeLocation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	registryPath := filepath.Join(s.extensionsPath, registryFileName)

	entries, err := readRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("register extension: %w", err)
	}

	entry := registryEntry{
		Identifier:       registryIdentifier{ID: id.String()},
		Version:          ver.String(),
		Location:         registryLocation{Mid: 1, Path: filepath.Join(s.extensionsPath, relativeLocation), Scheme: "file"},
		RelativeLocation: relativeLocation,
		Metadata:         json.RawMessage("{}"),
	}

	// Обновление существующей записи или добавление новой
	updated := false
	for i, e := range entries {
		if e.Identifier.ID == id.String() {
			entries[i] = entry
			updated = true
			break
		}
	}
	if !updated {
		entries = append(entries, entry)
	}

	return writeRegistry(registryPath, entries)
}

// Читает реестр расширений VS Code
func readRegistry(path string) ([]registryEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []registryEntry{}, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var entries []registryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	return entries, nil
}

// Записывает реестр расширений VS Code
func writeRegistry(path string, entries []registryEntry) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}

// Распаковывает vsix-пакет
func unpackVsix(zipReader *zip.Reader, destDir string) error {
	for _, f := range zipReader.File {
		relPath, found := strings.CutPrefix(f.Name, "extension/")
		if !found || relPath == "" {
			continue
		}
		targetPath := filepath.Join(destDir, relPath)
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			err := os.MkdirAll(targetPath, 0755)
			if err != nil {
				return fmt.Errorf("unpack vsix: %w", err)
			}
		} else {
			err := extractZipFile(f, targetPath)
			if err != nil {
				return fmt.Errorf("unpack vsix: %w", err)
			}
		}
	}
	return nil
}
