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
	"time"

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

// TODO возможно стоит брать расширение из реестра extensions.json
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
	extDirName := extDirName(id, version.Version)
	destDir := filepath.Join(s.extensionsPath, extDirName)
	zipReader, err := zip.NewReader(tmpFile, info.Size())
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if err := unpackVsix(zipReader, destDir); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if err := injectMetadata(destDir, version.Platform); err != nil {
		// Удаляем распакованное расширение при ошибке
		if rmErr := os.RemoveAll(destDir); rmErr != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, rmErr))
		}
		return fmt.Errorf("install: %w", err)
	}

	if err := s.registerExtension(id, version.Version, extDirName); err != nil {
		// Удаляем директорию расширения при ошибке регистрации в реестре
		if err := os.RemoveAll(destDir); err != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, err))
		}
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

// Remove удаляет расширение
func (s *VSCodeStorage) Remove(ctx context.Context, id domain.ExtensionID) error {
	ext, err := s.unregisterExtension(id)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	err = os.RemoveAll(ext.Location.Path)
	if err != nil {
		// Откатываем удаление из реестра vscode
		ver, parseErr := domain.ParseVersion(ext.Version)
		if parseErr != nil {
			s.logFunc(fmt.Sprintf("failed to parse version for rollback %s: %v", id.String(), parseErr))
			return fmt.Errorf("remove: %w", err)
		}
		if regErr := s.registerExtension(id, ver, ext.RelativeLocation); regErr != nil {
			s.logFunc(fmt.Sprintf("failed to rollback registry for %s: %v", id.String(), regErr))
		}
		return fmt.Errorf("remove: %w", err)
	}
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
	var extensionPack []domain.ExtensionID
	for _, ext := range pkg.ExtensionPack {
		id, err := domain.ParseExtensionID(ext)
		if err != nil {
			return domain.Extension{}, fmt.Errorf("parse extension dir: %w", err)
		}
		extensionPack = append(extensionPack, id)
	}

	return domain.Extension{
		ID: domain.ExtensionID{
			Name:      pkg.Name,
			Publisher: pkg.Publisher,
		},
		Description:   pkg.Description,
		Version:       version,
		Platform:      pkg.Metadata.TargetPlatform,
		ExtensionPack: extensionPack,
	}, nil
}

// Добавляет __metadata в package.json расширения
func injectMetadata(extDir string, platform domain.Platform) error {
	pkgPath := filepath.Join(extDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return fmt.Errorf("inject metadata: %w", err)
	}

	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("inject metadata: %w", err)
	}

	targetPlatform := string(platform)
	if targetPlatform == "" {
		targetPlatform = "undefined"
	}

	pkg["__metadata"] = map[string]any{
		"installedTimestamp": time.Now().UnixMilli(),
		"targetPlatform":     targetPlatform,
	}

	result, err := json.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("inject metadata: %w", err)
	}

	if err := os.WriteFile(pkgPath, result, 0644); err != nil {
		return fmt.Errorf("inject metadata: %w", err)
	}

	return nil
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

	registryPath := s.registryPath()

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

// Удаляет расширение из реестра VS Code (extensions.json)
func (s *VSCodeStorage) unregisterExtension(id domain.ExtensionID) (registryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	registryPath := s.registryPath()

	entries, err := readRegistry(registryPath)
	if err != nil {
		return registryEntry{}, fmt.Errorf("unregister extension: %w", err)
	}

	idx := -1
	for i, ext := range entries {
		if ext.Identifier.ID == id.String() {
			idx = i
			break
		}
	}

	if idx == -1 {
		return registryEntry{}, fmt.Errorf("unregister extension: %w", domain.ErrNotInstalled)
	}

	removedExt := entries[idx]
	entries = append(entries[:idx], entries[idx+1:]...)
	err = writeRegistry(registryPath, entries)
	if err != nil {
		return registryEntry{}, fmt.Errorf("unregister extension: %w", err)
	}

	return removedExt, nil
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
		// Извлекаем extension.vsixmanifest из корня архива как .vsixmanifest
		if f.Name == "extension.vsixmanifest" {
			targetPath := filepath.Join(destDir, ".vsixmanifest")
			if err := extractZipFile(f, targetPath); err != nil {
				return fmt.Errorf("unpack vsix: %w", err)
			}
			continue
		}

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

// Формирует наименование директории расширения
func extDirName(id domain.ExtensionID, ver domain.Version) string {
	return fmt.Sprintf("%s.%s-%s", id.Publisher, id.Name, ver.String())
}

// Формирует наименование файла реестра vscode
func (s *VSCodeStorage) registryPath() string {
	return filepath.Join(s.extensionsPath, registryFileName)
}
