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

const (
	registryFileName        = "extensions.json"
	undefinedTargetPlatform = "undefined"
)

type Storage struct {
	extensionsPath string
	logFunc        domain.LogFunc
	mu             sync.Mutex
}

func NewStorage(extensionsPath string, logFunc domain.LogFunc) *Storage {
	if logFunc == nil {
		logFunc = func(string) {}
	}
	return &Storage{
		extensionsPath: extensionsPath,
		logFunc:        logFunc,
	}
}

func (s *Storage) List(ctx context.Context) ([]domain.Extension, error) {
	info, err := os.Stat(s.extensionsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", s.extensionsPath, domain.ErrExtensionDirNotFound)
		}
		return nil, fmt.Errorf("list: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: %w", s.extensionsPath, domain.ErrExtensionDirNotFound)
	}

	entries, err := readRegistry(s.registryPath())
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	result := make([]domain.Extension, 0)

	for _, entry := range entries {
		dirPath := s.extPath(entry.RelativeLocation)
		extension, err := parseExtensionDir(dirPath)
		if err != nil {
			s.logFunc(fmt.Sprintf("failed to parse extension directory %s: %v", dirPath, err))
			continue
		}
		result = append(result, extension)
	}

	return result, nil
}

func (s *Storage) Install(ctx context.Context, params domain.InstallParams) error {
	// Запоминаем предыдущую директорию, если расширение уже установлено
	var previousDir string
	entries, err := readRegistry(s.registryPath())
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if idx := findEntryIndex(entries, params.ID); idx != -1 {
		previousDir = s.extPath(entries[idx].RelativeLocation)
	}

	tmpFile, err := saveToTempFile(params.Data)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	info, err := tmpFile.Stat()
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	dirName := extDirName(params.ID, params.Version, params.Platform)
	destDir := filepath.Join(s.extensionsPath, dirName)
	zipReader, err := zip.NewReader(tmpFile, info.Size())
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if err := unpackVsix(zipReader, destDir); err != nil {
		if rmErr := os.RemoveAll(destDir); rmErr != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, rmErr))
		}
		return fmt.Errorf("install: %w", err)
	}
	if err := injectMetadata(destDir, params.Platform, int64(len(params.Data))); err != nil {
		if rmErr := os.RemoveAll(destDir); rmErr != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, rmErr))
		}
		return fmt.Errorf("install: %w", err)
	}

	targetPlatform := string(params.Platform)
	if targetPlatform == "" {
		targetPlatform = undefinedTargetPlatform
	}

	metaJSON, err := json.Marshal(registryMetadata{
		InstalledTimestamp:   time.Now().UnixMilli(),
		Pinned:               false,
		Source:               "gallery",
		ID:                   params.Meta.UUID,
		PublisherID:          params.Meta.PublisherID,
		PublisherDisplayName: params.Meta.PublisherDisplayName,
		TargetPlatform:       targetPlatform,
		Updated:              false,
		IsPreReleaseVersion:  params.Meta.IsPreRelease,
		HasPreReleaseVersion: params.Meta.HasPreRelease,
	})
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}

	entry := registryEntry{
		Identifier:       registryIdentifier{ID: params.ID.String(), UUID: params.Meta.UUID},
		Version:          params.Version.String(),
		Location:         registryLocation{Mid: 1, Path: s.extPath(dirName), Scheme: "file"},
		RelativeLocation: dirName,
		Metadata:         metaJSON,
	}

	if err := s.registerExtension(entry); err != nil {
		if rmErr := os.RemoveAll(destDir); rmErr != nil {
			s.logFunc(fmt.Sprintf("failed to clean up %s: %v", destDir, rmErr))
		}
		return fmt.Errorf("install: %w", err)
	}

	// Удаляем директорию предыдущей версии, если она отличается от новой
	if previousDir != "" && previousDir != destDir {
		if rmErr := os.RemoveAll(previousDir); rmErr != nil {
			s.logFunc(fmt.Sprintf("failed to delete previous version %s: %v", previousDir, rmErr))
		}
	}

	return nil
}

// Remove удаляет расширение
func (s *Storage) Remove(ctx context.Context, id domain.ExtensionID) error {
	ext, err := s.unregisterExtension(id)
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	err = os.RemoveAll(s.extPath(ext.RelativeLocation))
	if err != nil {
		// Откатываем удаление из реестра vscode
		if regErr := s.registerExtension(ext); regErr != nil {
			s.logFunc(fmt.Sprintf("failed to rollback registry for %s: %v", id.String(), regErr))
		}
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func (s *Storage) IsInstalled(ctx context.Context, id domain.ExtensionID) (bool, error) {
	entries, err := readRegistry(s.registryPath())
	if err != nil {
		return false, fmt.Errorf("check installed: %w", err)
	}
	return findEntryIndex(entries, id) != -1, nil
}

func (s *Storage) InstalledVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	entries, err := readRegistry(s.registryPath())
	if err != nil {
		return domain.Version{}, fmt.Errorf("get installed version: %w", err)
	}
	idx := findEntryIndex(entries, id)
	if idx == -1 {
		return domain.Version{}, fmt.Errorf("get installed version: %w", domain.ErrNotInstalled)
	}
	ver, err := domain.ParseVersion(entries[idx].Version)
	if err != nil {
		return domain.Version{}, fmt.Errorf("get installed version: %w", err)
	}
	return ver, nil
}

func (s *Storage) Update(ctx context.Context, params domain.InstallParams) error {
	entries, err := readRegistry(s.registryPath())
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	idx := findEntryIndex(entries, params.ID)
	if idx == -1 {
		return fmt.Errorf("update: %w", domain.ErrNotInstalled)
	}

	installedVersion, err := domain.ParseVersion(entries[idx].Version)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if installedVersion == params.Version {
		return fmt.Errorf("update: %w", domain.ErrAlreadyInstalled)
	}

	// Install сам удалит директорию предыдущей версии
	if err := s.Install(ctx, params); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

// Извлекает информацию о расширении из его директории
// Для этого парсит файл package.json
func parseExtensionDir(dirPath string) (domain.Extension, error) {
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
	id, err := domain.ParseExtensionID(pkg.Publisher + "." + pkg.Name)
	if err != nil {
		return domain.Extension{}, fmt.Errorf("parse extension dir: %w", err)
	}
	return domain.Extension{
		ID:            id,
		Description:   pkg.Description,
		Version:       version,
		Platform:      pkg.Metadata.TargetPlatform,
		ExtensionPack: extensionPack,
		Size:          pkg.Metadata.Size,
	}, nil
}

// Добавляет __metadata в package.json расширения
func injectMetadata(extDir string, platform domain.Platform, size int64) error {
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
		targetPlatform = undefinedTargetPlatform
	}

	pkg["__metadata"] = map[string]any{
		"installedTimestamp": time.Now().UnixMilli(),
		"targetPlatform":     targetPlatform,
		"size":               size,
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
func (s *Storage) registerExtension(newEntry registryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	registryPath := s.registryPath()

	entries, err := readRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	// Обновление существующей записи или добавление новой
	idx := -1
	for i, e := range entries {
		if e.Identifier.ID == newEntry.Identifier.ID {
			idx = i
			break
		}
	}
	if idx != -1 {
		entries[idx] = newEntry
	} else {
		entries = append(entries, newEntry)
	}

	return writeRegistry(registryPath, entries)
}

// Удаляет расширение из реестра VS Code (extensions.json)
func (s *Storage) unregisterExtension(id domain.ExtensionID) (registryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	registryPath := s.registryPath()

	entries, err := readRegistry(registryPath)
	if err != nil {
		return registryEntry{}, fmt.Errorf("unregister: %w", err)
	}

	idx := findEntryIndex(entries, id)
	if idx == -1 {
		return registryEntry{}, fmt.Errorf("unregister: %w", domain.ErrNotInstalled)
	}

	removedExt := entries[idx]
	entries = append(entries[:idx], entries[idx+1:]...)
	err = writeRegistry(registryPath, entries)
	if err != nil {
		return registryEntry{}, fmt.Errorf("unregister: %w", err)
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

// Ищет запись расширения в реестре по ID, возвращает индекс или -1
func findEntryIndex(entries []registryEntry, id domain.ExtensionID) int {
	for i, entry := range entries {
		if entry.Identifier.ID == id.String() {
			return i
		}
	}
	return -1
}

// Формирует наименование директории расширения
func extDirName(id domain.ExtensionID, ver domain.Version, platform domain.Platform) string {
	name := fmt.Sprintf("%s.%s-%s", id.Publisher, id.Name, ver.String())
	if platform != "" {
		name += "-" + string(platform)
	}
	return name
}

// Возвращает абсолютный путь к директории расширения по относительному расположению
func (s *Storage) extPath(relativeLocation string) string {
	return filepath.Join(s.extensionsPath, relativeLocation)
}

// Формирует наименование файла реестра vscode
func (s *Storage) registryPath() string {
	return filepath.Join(s.extensionsPath, registryFileName)
}
