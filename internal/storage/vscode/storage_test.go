package vscode

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// errAny используется в табличных тестах, когда важен сам факт ошибки, а не конкретный тип
var errAny = errors.New("any error")

func TestParseExtensionDir(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string // содержимое package.json ("" - не создавать файл)
		want        domain.Extension
		wantErr     bool
	}{
		{
			name:        "valid_extension",
			packageJSON: `{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`,
			want: domain.Extension{
				ID: domain.ExtensionID{
					Publisher: "golang",
					Name:      "go",
				},
				Description: "Go support",
				Version: domain.Version{
					Major: 0,
					Minor: 53,
					Patch: 1,
				},
			},
		},
		{
			name:        "with_platform",
			packageJSON: `{"publisher":"ms-python","name":"debugpy","version":"2025.18.0","__metadata":{"targetPlatform":"linux-x64"}}`,
			want: domain.Extension{
				ID: domain.ExtensionID{
					Publisher: "ms-python",
					Name:      "debugpy",
				},
				Version: domain.Version{
					Major: 2025,
					Minor: 18,
					Patch: 0,
				},
				Platform: domain.LinuxX64,
			},
		},
		{
			name:        "with_size",
			packageJSON: `{"publisher":"golang","name":"go","version":"0.53.1","__metadata":{"size":2954467}}`,
			want: domain.Extension{
				ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
				Version: domain.Version{Major: 0, Minor: 53, Patch: 1},
				Size:    2954467,
			},
		},
		{
			name:        "no_package_json",
			packageJSON: "",
			wantErr:     true,
		},
		{
			name:        "invalid_json",
			packageJSON: `{not json}`,
			wantErr:     true,
		},
		{
			name:        "with_extension_pack",
			packageJSON: `{"publisher":"vue","name":"volar","version":"2.2.2","extensionPack":["vue.typescript-plugin","dbaeumer.vscode-eslint"]}`,
			want: domain.Extension{
				ID:      domain.ExtensionID{Publisher: "vue", Name: "volar"},
				Version: domain.Version{Major: 2, Minor: 2, Patch: 2},
				ExtensionPack: []domain.ExtensionID{
					{Publisher: "vue", Name: "typescript-plugin"},
					{Publisher: "dbaeumer", Name: "vscode-eslint"},
				},
			},
		},
		{
			name:        "invalid_extension_pack_id",
			packageJSON: `{"publisher":"test","name":"ext","version":"1.0.0","extensionPack":["invalid-id"]}`,
			wantErr:     true,
		},
		{
			name:        "invalid_version",
			packageJSON: `{"publisher":"test","name":"ext","version":"abc"}`,
			wantErr:     true,
		},
		{
			name:        "to_lower_case",
			packageJSON: `{"publisher":"Golang","name":"Go","version":"0.53.1","description":"Go support"}`,
			want: domain.Extension{
				ID: domain.ExtensionID{
					Publisher: "golang",
					Name:      "go",
				},
				Description: "Go support",
				Version: domain.Version{
					Major: 0,
					Minor: 53,
					Patch: 1,
				},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()

			if testCase.packageJSON != "" {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(testCase.packageJSON), 0o644)
			}

			got, err := parseExtensionDir(dir)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name          string
		setupExtDir   func(dir string)
		setupRegistry func(dir string) string // Формирует JSON реестра с реальными путями
		want          []domain.Extension
		wantErr       error
	}{
		{
			name: "multiple_extensions",
			setupExtDir: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
				writePackageJSON(t, dir, "ms-python.python-2026.2.0",
					`{"publisher":"ms-python","name":"python","version":"2026.2.0","description":"Python support"}`)
			},
			setupRegistry: func(dir string) string {
				return fmt.Sprintf(`[
				{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"},
				{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"%s/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0","metadata":{"publisherDisplayName":"Microsoft","installedTimestamp":1770717444996}}
				]`, dir, dir)
			},
			want: []domain.Extension{
				{
					ID:          domain.ExtensionID{Publisher: "golang", Name: "go"},
					Description: "Go support",
					Version:     domain.Version{Major: 0, Minor: 53, Patch: 1},
				},
				{
					ID:          domain.ExtensionID{Publisher: "ms-python", Name: "python"},
					Description: "Python support",
					Version:     domain.Version{Major: 2026, Minor: 2, Patch: 0},
				},
			},
		},
		{
			name:        "empty_registry",
			setupExtDir: func(dir string) {},
			want:        []domain.Extension{},
		},
		{
			name: "skips_broken_extensions",
			setupExtDir: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
				writePackageJSON(t, dir, "golang.broken-0.53.1",
					`{"publisher":"golang","name":"broken","version":"0.53.1","description":"Go support"`)
			},
			setupRegistry: func(dir string) string {
				return fmt.Sprintf(`[
				{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"},
				{"identifier":{"id":"golang.broken"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.broken-0.53.1","scheme":"file"},"relativeLocation":"golang.broken-0.53.1"}
				]`, dir, dir)
			},
			want: []domain.Extension{
				{
					ID:          domain.ExtensionID{Publisher: "golang", Name: "go"},
					Description: "Go support",
					Version:     domain.Version{Major: 0, Minor: 53, Patch: 1},
				},
			},
		},
		{
			name:          "directory_not_exists",
			setupExtDir:   nil,
			setupRegistry: nil,
			wantErr:       domain.ErrExtensionDirNotFound,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.setupExtDir != nil {
				testCase.setupExtDir(dir)
			}
			path := filepath.Join(dir, registryFileName)
			if testCase.setupRegistry != nil {
				os.WriteFile(path, []byte(testCase.setupRegistry(dir)), 0o644)
			}

			storagePath := dir
			if testCase.wantErr != nil {
				storagePath = filepath.Join(dir, "nonexistent")
			}

			storage := NewStorage(storagePath, nil)
			got, err := storage.List(context.Background())

			if testCase.wantErr != nil && err == nil {
				t.Fatal("expected error, got nil")
			}
			if testCase.wantErr != nil && err != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("expected error: %v, got error: %v", testCase.wantErr, err)
				}
			}
			if testCase.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.wantErr == nil && !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestListLogsBrokenExtensions(t *testing.T) {
	dir := t.TempDir()

	writePackageJSON(t, dir, "golang.go-0.53.1",
		`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
	writePackageJSON(t, dir, "golang.broken-0.53.1",
		`{"publisher":"golang","name":"broken","version":"0.53.1"`)

	registry := fmt.Sprintf(`[
		{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"},
		{"identifier":{"id":"golang.broken"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.broken-0.53.1","scheme":"file"},"relativeLocation":"golang.broken-0.53.1"}
	]`, dir, dir)
	os.WriteFile(filepath.Join(dir, registryFileName), []byte(registry), 0o644)

	var logMessages []string
	logFunc := func(msg string) { logMessages = append(logMessages, msg) }

	storage := NewStorage(dir, logFunc)
	got, err := storage.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d extensions, want 1", len(got))
	}
	if got[0].ID.Name != "go" {
		t.Errorf("got extension %s, want go", got[0].ID.Name)
	}
	if len(logMessages) != 1 {
		t.Fatalf("got %d log messages, want 1", len(logMessages))
	}
	if !strings.Contains(logMessages[0], "golang.broken-0.53.1") {
		t.Errorf("log message doesn't mention broken extension: %s", logMessages[0])
	}
}

func TestListPathIsFileNotDirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	os.WriteFile(filePath, []byte(""), 0o644)

	storage := NewStorage(filePath, nil)
	_, err := storage.List(context.Background())

	if !errors.Is(err, domain.ErrExtensionDirNotFound) {
		t.Fatalf("got %v, want %v", err, domain.ErrExtensionDirNotFound)
	}
}

func TestInstall(t *testing.T) {
	tests := []struct {
		name      string
		zipFiles  map[string]string // путь в архиве: содержимое
		wantFiles []string          // ожидаемые файлы относительно destDir
		wantErr   bool
	}{
		{
			name: "happy_path",
			zipFiles: map[string]string{
				"extension/package.json": `{"name":"go"}`,
				"extension/main.js":      "console.log('hello')",
			},
			wantFiles: []string{
				"package.json",
				"main.js",
			},
		},
		{
			name: "skips_non_extension_files",
			zipFiles: map[string]string{
				"[Content_Types].xml":    "<xml/>",
				"extension/package.json": `{"name":"go"}`,
			},
			wantFiles: []string{
				"package.json",
			},
		},
		{
			name: "nested_directories",
			zipFiles: map[string]string{
				"extension/src/lib/utils.js": "export default {}",
				"extension/package.json":     `{"name":"go"}`,
			},
			wantFiles: []string{
				"package.json",
				"src/lib/utils.js",
			},
		},
		{
			name: "extracts_vsixmanifest",
			zipFiles: map[string]string{
				"extension.vsixmanifest": "<xml>manifest</xml>",
				"extension/package.json": `{"name":"go"}`,
				"[Content_Types].xml":    "<xml/>",
			},
			wantFiles: []string{
				"package.json",
				".vsixmanifest",
			},
		},
		{
			name: "path_traversal",
			zipFiles: map[string]string{
				"extension/../../etc/passwd": "root:x:0:0",
				"extension/package.json":     `{"name":"go"}`,
				"extension/safe.js":          "ok",
			},
			wantFiles: []string{
				"package.json",
				"safe.js",
			},
		},
	}

	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	version := domain.Version{Major: 1, Minor: 0, Patch: 0}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			storage := NewStorage(dir, nil)

			vsix := createZip(t, testCase.zipFiles)
			err := storage.Install(context.Background(), domain.InstallParams{
				ID: id, Version: version, Data: vsix.Bytes(),
			})

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.wantErr {
				return
			}

			destDir := filepath.Join(dir, "golang.go-1.0.0")
			for _, wantFile := range testCase.wantFiles {
				path := filepath.Join(destDir, wantFile)
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("file %s: %v", wantFile, err)
				}

				// .vsixmanifest извлекается из корня архива - проверяем содержимое
				if wantFile == ".vsixmanifest" {
					want := testCase.zipFiles["extension.vsixmanifest"]
					if string(got) != want {
						t.Errorf("file %s: got %q, want %q", wantFile, string(got), want)
					}
					continue
				}

				// package.json модифицируется injectMetadata - проверяем содержимое __metadata
				if wantFile == "package.json" {
					var pkg map[string]any
					if err := json.Unmarshal(got, &pkg); err != nil {
						t.Fatalf("file %s: invalid json: %v", wantFile, err)
					}
					meta, ok := pkg["__metadata"].(map[string]any)
					if !ok {
						t.Fatalf("file %s: __metadata is not an object", wantFile)
					}
					if _, ok := meta["installedTimestamp"]; !ok {
						t.Errorf("file %s: missing installedTimestamp in __metadata", wantFile)
					}
					if _, ok := meta["targetPlatform"]; !ok {
						t.Errorf("file %s: missing targetPlatform in __metadata", wantFile)
					}
					if _, ok := meta["size"]; !ok {
						t.Errorf("file %s: missing size in __metadata", wantFile)
					}
					continue
				}

				want, ok := testCase.zipFiles["extension/"+wantFile]
				if !ok {
					t.Fatalf("file %s: not found in zipFiles", wantFile)
				}
				if string(got) != want {
					t.Errorf("file %s: got %q, want %q", wantFile, string(got), want)
				}
			}
		})
	}
}

func TestInstallInvalidZip(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	err := storage.Install(context.Background(), domain.InstallParams{
		ID:      domain.ExtensionID{Publisher: "test", Name: "ext"},
		Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
		Data:    []byte("not a zip"),
	})
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

// createZip - хелпер для создания zip-архива в памяти
func createZip(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return &buf
}

func TestExtractZipFilePreservesPermissions(t *testing.T) {
	tests := []struct {
		name     string
		fileMode os.FileMode
	}{
		{
			name:     "executable_file",
			fileMode: 0o755,
		},
		{
			name:     "regular_file",
			fileMode: 0o644,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			header := &zip.FileHeader{Name: "extension/bin/tool"}
			header.SetMode(testCase.fileMode)
			f, err := w.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			f.Write([]byte("binary"))
			w.Close()

			reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				t.Fatal(err)
			}

			destDir := t.TempDir()
			targetPath := filepath.Join(destDir, "bin", "tool")
			err = extractZipFile(reader.File[0], targetPath)
			if err != nil {
				t.Fatal(err)
			}

			info, err := os.Stat(targetPath)
			if err != nil {
				t.Fatal(err)
			}
			got := info.Mode().Perm()
			if got != testCase.fileMode {
				t.Errorf("got permissions %o, want %o", got, testCase.fileMode)
			}
		})
	}
}

// writePackageJSON - хелпер для создания директории расширения с package.json
func writePackageJSON(t *testing.T, baseDir, extDir, content string) {
	t.Helper()
	dir := filepath.Join(baseDir, extDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadRegistry(t *testing.T) {
	tests := []struct {
		name      string
		content   string // "" - файл не создаётся
		wantCount int
		wantErr   bool
	}{
		{
			name:      "file_not_exists",
			content:   "",
			wantCount: 0,
		},
		{
			name:      "empty_array",
			content:   "[]",
			wantCount: 0,
		},
		{
			name:      "single_entry",
			content:   `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`,
			wantCount: 1,
		},
		{
			name:      "multiple_entries",
			content:   `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"},{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"/ext/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0"}]`,
			wantCount: 2,
		},
		{
			name:    "invalid_json",
			content: "{not json}",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, registryFileName)

			if testCase.content != "" {
				os.WriteFile(path, []byte(testCase.content), 0o644)
			}

			entries, err := readRegistry(path)

			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(entries) != testCase.wantCount {
				t.Errorf("got %d entries, want %d", len(entries), testCase.wantCount)
			}
		})
	}
}

func TestRegisterExtension(t *testing.T) {
	tests := []struct {
		name             string
		existingRegistry string // "" - файл не существует
		entry            registryEntry
		wantCount        int
		wantErr          bool
	}{
		{
			name:             "no_existing_file",
			existingRegistry: "",
			entry: registryEntry{
				Identifier:       registryIdentifier{ID: "golang.go"},
				Version:          "0.53.1",
				Location:         registryLocation{Mid: 1, Scheme: "file"},
				RelativeLocation: "golang.go-0.53.1",
				Metadata:         json.RawMessage("{}"),
			},
			wantCount: 1,
		},
		{
			name:             "empty_registry",
			existingRegistry: "[]",
			entry: registryEntry{
				Identifier:       registryIdentifier{ID: "golang.go"},
				Version:          "0.53.1",
				Location:         registryLocation{Mid: 1, Scheme: "file"},
				RelativeLocation: "golang.go-0.53.1",
				Metadata:         json.RawMessage("{}"),
			},
			wantCount: 1,
		},
		{
			name:             "append_to_existing",
			existingRegistry: `[{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"/ext/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0"}]`,
			entry: registryEntry{
				Identifier:       registryIdentifier{ID: "golang.go"},
				Version:          "0.53.1",
				Location:         registryLocation{Mid: 1, Scheme: "file"},
				RelativeLocation: "golang.go-0.53.1",
				Metadata:         json.RawMessage("{}"),
			},
			wantCount: 2,
		},
		{
			name:             "update_existing",
			existingRegistry: `[{"identifier":{"id":"golang.go"},"version":"0.52.0","location":{"$mid":1,"path":"/ext/golang.go-0.52.0","scheme":"file"},"relativeLocation":"golang.go-0.52.0"}]`,
			entry: registryEntry{
				Identifier:       registryIdentifier{ID: "golang.go"},
				Version:          "0.53.1",
				Location:         registryLocation{Mid: 1, Scheme: "file"},
				RelativeLocation: "golang.go-0.53.1",
				Metadata:         json.RawMessage("{}"),
			},
			wantCount: 1,
		},
		{
			name:             "invalid_json",
			existingRegistry: "{not json}",
			entry: registryEntry{
				Identifier:       registryIdentifier{ID: "golang.go"},
				Version:          "0.53.1",
				Location:         registryLocation{Mid: 1, Scheme: "file"},
				RelativeLocation: "golang.go-0.53.1",
				Metadata:         json.RawMessage("{}"),
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			storage := NewStorage(dir, nil)

			if testCase.existingRegistry != "" {
				os.WriteFile(filepath.Join(dir, registryFileName), []byte(testCase.existingRegistry), 0o644)
			}

			// Заполняем path с реальной директорией
			entry := testCase.entry
			entry.Location.Path = filepath.Join(dir, entry.RelativeLocation)

			err := storage.registerExtension(entry)

			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			entries, err := readRegistry(filepath.Join(dir, registryFileName))
			if err != nil {
				t.Fatalf("failed to read registry: %v", err)
			}

			if len(entries) != testCase.wantCount {
				t.Fatalf("got %d entries, want %d", len(entries), testCase.wantCount)
			}

			// Проверяем запись нашего расширения
			var found *registryEntry
			for i, e := range entries {
				if e.Identifier.ID == entry.Identifier.ID {
					found = &entries[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("entry %s not found in registry", entry.Identifier.ID)
			}
			if found.Version != entry.Version {
				t.Errorf("version: got %s, want %s", found.Version, entry.Version)
			}
			if found.RelativeLocation != entry.RelativeLocation {
				t.Errorf("relativeLocation: got %s, want %s", found.RelativeLocation, entry.RelativeLocation)
			}
			if found.Location.Path != entry.Location.Path {
				t.Errorf("location.path: got %s, want %s", found.Location.Path, entry.Location.Path)
			}
			if found.Location.Scheme != "file" {
				t.Errorf("location.scheme: got %s, want file", found.Location.Scheme)
			}
			if found.Location.Mid != 1 {
				t.Errorf("location.$mid: got %d, want 1", found.Location.Mid)
			}
		})
	}
}

func TestRegisterExtensionPreservesMetadata(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	existingJSON := `[{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"/ext/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0","metadata":{"publisherDisplayName":"Microsoft","installedTimestamp":1770717444996}}]`
	os.WriteFile(filepath.Join(dir, registryFileName), []byte(existingJSON), 0o644)

	entry := registryEntry{
		Identifier:       registryIdentifier{ID: "golang.go"},
		Version:          "0.53.1",
		Location:         registryLocation{Mid: 1, Path: filepath.Join(dir, "golang.go-0.53.1"), Scheme: "file"},
		RelativeLocation: "golang.go-0.53.1",
		Metadata:         json.RawMessage("{}"),
	}
	if err := storage.registerExtension(entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := readRegistry(filepath.Join(dir, registryFileName))
	if err != nil {
		t.Fatalf("failed to read registry: %v", err)
	}

	// Находим существующую запись и проверяем что метадата сохранилась
	for _, e := range entries {
		if e.Identifier.ID == "ms-python.python" {
			var metadata map[string]any
			if err := json.Unmarshal(e.Metadata, &metadata); err != nil {
				t.Fatalf("failed to unmarshal metadata: %v", err)
			}
			if metadata["publisherDisplayName"] != "Microsoft" {
				t.Errorf("metadata publisherDisplayName: got %v, want Microsoft", metadata["publisherDisplayName"])
			}
			if metadata["installedTimestamp"] != float64(1770717444996) {
				t.Errorf("metadata installedTimestamp: got %v, want 1770717444996", metadata["installedTimestamp"])
			}
			return
		}
	}
	t.Fatal("existing entry ms-python.python not found")
}

func TestInstallCreatesRegistryEntry(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	version := domain.Version{Major: 1, Minor: 0, Patch: 0}
	meta := domain.ExtensionMeta{
		UUID:                 "test-uuid-123",
		PublisherID:          "pub-uuid-456",
		PublisherDisplayName: "Go Team at Google",
	}

	vsix := createZip(t, map[string]string{
		"extension/package.json": `{"name":"go"}`,
	})
	err := storage.Install(context.Background(), domain.InstallParams{
		ID: id, Version: version, Meta: meta, Data: vsix.Bytes(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := readRegistry(filepath.Join(dir, registryFileName))
	if err != nil {
		t.Fatalf("failed to read registry: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Identifier.ID != "golang.go" {
		t.Errorf("identifier.id: got %s, want golang.go", entries[0].Identifier.ID)
	}
	if entries[0].Identifier.UUID != "test-uuid-123" {
		t.Errorf("identifier.uuid: got %s, want test-uuid-123", entries[0].Identifier.UUID)
	}
	if entries[0].Version != "1.0.0" {
		t.Errorf("version: got %s, want 1.0.0", entries[0].Version)
	}
	if entries[0].RelativeLocation != "golang.go-1.0.0" {
		t.Errorf("relativeLocation: got %s, want golang.go-1.0.0", entries[0].RelativeLocation)
	}
	wantPath := filepath.Join(dir, "golang.go-1.0.0")
	if entries[0].Location.Path != wantPath {
		t.Errorf("location.path: got %s, want %s", entries[0].Location.Path, wantPath)
	}

	// Проверяем метаданные
	var metadata map[string]any
	if err := json.Unmarshal(entries[0].Metadata, &metadata); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}
	if metadata["id"] != "test-uuid-123" {
		t.Errorf("metadata.id: got %v, want test-uuid-123", metadata["id"])
	}
	if metadata["publisherId"] != "pub-uuid-456" {
		t.Errorf("metadata.publisherId: got %v, want pub-uuid-456", metadata["publisherId"])
	}
	if metadata["publisherDisplayName"] != "Go Team at Google" {
		t.Errorf("metadata.publisherDisplayName: got %v, want Go Team at Google", metadata["publisherDisplayName"])
	}
	if metadata["source"] != "gallery" {
		t.Errorf("metadata.source: got %v, want gallery", metadata["source"])
	}
	if metadata["targetPlatform"] != "undefined" {
		t.Errorf("metadata.targetPlatform: got %v, want undefined", metadata["targetPlatform"])
	}
	if _, ok := metadata["installedTimestamp"]; !ok {
		t.Error("metadata.installedTimestamp: missing")
	}
}

func TestInstallWithPlatformSuffix(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	id := domain.ExtensionID{Publisher: "ms-python", Name: "debugpy"}
	version := domain.Version{Major: 2025, Minor: 18, Patch: 0}

	vsix := createZip(t, map[string]string{
		"extension/package.json": `{"name":"debugpy"}`,
	})
	err := storage.Install(context.Background(), domain.InstallParams{
		ID: id, Version: version, Platform: domain.LinuxX64, Data: vsix.Bytes(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Директория содержит platform-суффикс
	wantDir := filepath.Join(dir, "ms-python.debugpy-2025.18.0-linux-x64")
	if _, err := os.Stat(wantDir); err != nil {
		t.Errorf("platform dir not found: %v", err)
	}

	// Реестр содержит platform-суффикс
	entries, err := readRegistry(filepath.Join(dir, registryFileName))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].RelativeLocation != "ms-python.debugpy-2025.18.0-linux-x64" {
		t.Errorf("relativeLocation: got %s, want ms-python.debugpy-2025.18.0-linux-x64", entries[0].RelativeLocation)
	}
	if entries[0].Location.Path != wantDir {
		t.Errorf("location.path: got %s, want %s", entries[0].Location.Path, wantDir)
	}

	// targetPlatform в метаданных
	var metadata map[string]any
	if err := json.Unmarshal(entries[0].Metadata, &metadata); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}
	if metadata["targetPlatform"] != "linux-x64" {
		t.Errorf("metadata.targetPlatform: got %v, want linux-x64", metadata["targetPlatform"])
	}
}

func TestInstallOverExisting(t *testing.T) {
	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	oldVersion := domain.Version{Major: 1, Minor: 0, Patch: 0}
	newVersion := domain.Version{Major: 2, Minor: 0, Patch: 0}

	// Хелпер: устанавливает расширение
	installVersion := func(t *testing.T, storage *Storage, ver domain.Version) {
		t.Helper()
		vsix := createZip(t, map[string]string{
			"extension/package.json": fmt.Sprintf(`{"publisher":"golang","name":"go","version":"%s"}`, ver),
		})
		if err := storage.Install(context.Background(), domain.InstallParams{
			ID: id, Version: ver, Data: vsix.Bytes(),
		}); err != nil {
			t.Fatalf("install v%s: %v", ver, err)
		}
	}

	t.Run("removes_old_directory", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewStorage(dir, nil)

		installVersion(t, storage, oldVersion)
		installVersion(t, storage, newVersion)

		// Новая директория существует
		newDir := filepath.Join(dir, "golang.go-2.0.0")
		if _, err := os.Stat(newDir); err != nil {
			t.Errorf("new version dir not found: %v", err)
		}

		// Старая директория удалена
		oldDir := filepath.Join(dir, "golang.go-1.0.0")
		if _, err := os.Stat(oldDir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("old version dir still exists")
		}

		// Реестр содержит новую версию
		entries, err := readRegistry(filepath.Join(dir, registryFileName))
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Version != newVersion.String() {
			t.Errorf("registry version: got %s, want %s", entries[0].Version, newVersion.String())
		}
	})

	t.Run("same_version_overwrites", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewStorage(dir, nil)

		installVersion(t, storage, oldVersion)
		installVersion(t, storage, oldVersion)

		// Директория на месте
		extDir := filepath.Join(dir, "golang.go-1.0.0")
		if _, err := os.Stat(extDir); err != nil {
			t.Errorf("dir not found: %v", err)
		}

		// Одна запись в реестре
		entries, err := readRegistry(filepath.Join(dir, registryFileName))
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
	})

	t.Run("failure_preserves_old_version", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewStorage(dir, nil)

		installVersion(t, storage, oldVersion)

		// Пытаемся установить новую версию с невалидным zip
		err := storage.Install(context.Background(), domain.InstallParams{
			ID: id, Version: newVersion, Data: []byte("not a zip"),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Старая директория на месте
		oldDir := filepath.Join(dir, "golang.go-1.0.0")
		if _, err := os.Stat(oldDir); err != nil {
			t.Errorf("old version dir should still exist: %v", err)
		}

		// Реестр не изменился
		entries, err := readRegistry(filepath.Join(dir, registryFileName))
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Version != oldVersion.String() {
			t.Errorf("registry version: got %s, want %s", entries[0].Version, oldVersion.String())
		}
	})
}

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name     string
		registry string // содержимое extensions.json ("" - файл не создаётся)
		id       domain.ExtensionID
		want     bool
	}{
		{
			name:     "installed",
			registry: `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`,
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			want:     true,
		},
		{
			name:     "not_installed",
			registry: `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`,
			id:       domain.ExtensionID{Publisher: "ms-python", Name: "python"},
			want:     false,
		},
		{
			name:     "empty_registry",
			registry: `[]`,
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			want:     false,
		},
		{
			name:     "no_registry_file",
			registry: "",
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			want:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.registry != "" {
				os.WriteFile(filepath.Join(dir, registryFileName), []byte(testCase.registry), 0o644)
			}

			storage := NewStorage(dir, nil)
			got, err := storage.IsInstalled(context.Background(), testCase.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Errorf("got %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestInstalledVersion(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		id       domain.ExtensionID
		want     domain.Version
		wantErr  error
	}{
		{
			name:     "found",
			registry: `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`,
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			want:     domain.Version{Major: 0, Minor: 53, Patch: 1},
		},
		{
			name:     "not_installed",
			registry: `[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"/ext/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`,
			id:       domain.ExtensionID{Publisher: "ms-python", Name: "python"},
			wantErr:  domain.ErrNotInstalled,
		},
		{
			name:     "empty_registry",
			registry: `[]`,
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			wantErr:  domain.ErrNotInstalled,
		},
		{
			name:     "invalid_version_in_registry",
			registry: `[{"identifier":{"id":"golang.go"},"version":"not-a-version","location":{"$mid":1,"path":"/ext/golang.go","scheme":"file"},"relativeLocation":"golang.go"}]`,
			id:       domain.ExtensionID{Publisher: "golang", Name: "go"},
			wantErr:  errAny,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, registryFileName), []byte(testCase.registry), 0o644)

			storage := NewStorage(dir, nil)
			got, err := storage.InstalledVersion(context.Background(), testCase.id)

			if testCase.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if testCase.wantErr != errAny && !errors.Is(err, testCase.wantErr) {
					t.Fatalf("got error %v, want %v", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Errorf("got %s, want %s", got, testCase.want)
			}
		})
	}
}

func TestRegisterExtensionConcurrent(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	const count = 50
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			relLoc := fmt.Sprintf("pub.ext-%d-1.0.%d", i, i)
			entry := registryEntry{
				Identifier:       registryIdentifier{ID: fmt.Sprintf("pub.ext-%d", i)},
				Version:          fmt.Sprintf("1.0.%d", i),
				Location:         registryLocation{Mid: 1, Path: filepath.Join(dir, relLoc), Scheme: "file"},
				RelativeLocation: relLoc,
				Metadata:         json.RawMessage("{}"),
			}
			if err := storage.registerExtension(entry); err != nil {
				t.Errorf("registerExtension(%s): %v", entry.Identifier.ID, err)
			}
		})
	}
	wg.Wait()

	entries, err := readRegistry(filepath.Join(dir, registryFileName))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if len(entries) != count {
		t.Errorf("got %d entries, want %d", len(entries), count)
	}
}

func TestUnregisterExtensionConcurrent(t *testing.T) {
	dir := t.TempDir()
	storage := NewStorage(dir, nil)

	const count = 50
	// Регистрируем расширения последовательно
	for i := range count {
		relLoc := fmt.Sprintf("pub.ext-%d-1.0.%d", i, i)
		entry := registryEntry{
			Identifier:       registryIdentifier{ID: fmt.Sprintf("pub.ext-%d", i)},
			Version:          fmt.Sprintf("1.0.%d", i),
			Location:         registryLocation{Mid: 1, Path: filepath.Join(dir, relLoc), Scheme: "file"},
			RelativeLocation: relLoc,
			Metadata:         json.RawMessage("{}"),
		}
		if err := storage.registerExtension(entry); err != nil {
			t.Fatalf("setup registerExtension(%d): %v", i, err)
		}
	}

	// Удаляем параллельно
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			id := domain.ExtensionID{Publisher: "pub", Name: fmt.Sprintf("ext-%d", i)}
			if _, err := storage.unregisterExtension(id); err != nil {
				t.Errorf("unregisterExtension(%s): %v", id, err)
			}
		})
	}
	wg.Wait()

	entries, err := readRegistry(filepath.Join(dir, registryFileName))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name          string
		setupRegistry func(dir string) string // Формирует JSON реестра с реальными путями
		setupExtDir   func(dir string)        // Создаём моковые директории с расширениями
		id            domain.ExtensionID
		wantCount     int
		wantErr       error
	}{
		{
			name: "simple_delete",
			setupRegistry: func(dir string) string {
				return fmt.Sprintf(`[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`, dir)
			},
			setupExtDir: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
			},
			id:        domain.ExtensionID{Publisher: "golang", Name: "go"},
			wantCount: 0,
		},
		{
			name: "delete_with_other_extensions",
			setupRegistry: func(dir string) string {
				return fmt.Sprintf(`[
				{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"},
				{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"%s/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0","metadata":{"publisherDisplayName":"Microsoft","installedTimestamp":1770717444996}}
				]`, dir, dir)
			},
			setupExtDir: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)

				writePackageJSON(t, dir, "ms-python.python-2026.2.0",
					`{"publisher":"ms-python","name":"python","version":"2026.2.0","description":"Go support"}`)
			},
			id:        domain.ExtensionID{Publisher: "ms-python", Name: "python"},
			wantCount: 1,
		},
		{
			name: "delete_not_installed_extension",
			setupRegistry: func(dir string) string {
				return fmt.Sprintf(`[{"identifier":{"id":"golang.go"},"version":"0.53.1","location":{"$mid":1,"path":"%s/golang.go-0.53.1","scheme":"file"},"relativeLocation":"golang.go-0.53.1"}]`, dir)
			},
			setupExtDir: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
			},
			id:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
			wantErr: domain.ErrNotInstalled,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.setupExtDir != nil {
				testCase.setupExtDir(dir)
			}
			path := filepath.Join(dir, registryFileName)
			if testCase.setupRegistry != nil {
				os.WriteFile(path, []byte(testCase.setupRegistry(dir)), 0o644)
			}

			storage := NewStorage(dir, nil)
			err := storage.Remove(t.Context(), testCase.id)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Errorf("got %v, want %v", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Проверяем кол-во расширений в реестре vscode
			entries, err := readRegistry(filepath.Join(dir, registryFileName))
			if err != nil {
				t.Fatalf("failed to read registry: %v", err)
			}
			if len(entries) != testCase.wantCount {
				t.Fatalf("got %d entries, want %d", len(entries), testCase.wantCount)
			}

			// Проверяем кол-во расширений в директории с расширениями
			dirEntries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("failed to read dir: %v", err)
			}
			var count int
			for _, entry := range dirEntries {
				if entry.IsDir() {
					count++
				}
			}

			if count != testCase.wantCount {
				t.Fatalf("got %d extensions, want %d", count, testCase.wantCount)
			}

		})
	}
}

func TestUpdate(t *testing.T) {
	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	oldVersion := domain.Version{Major: 0, Minor: 52, Patch: 0}
	newVersion := domain.Version{Major: 0, Minor: 53, Patch: 1}

	// Хелпер: устанавливает расширение старой версии
	installOldVersion := func(t *testing.T, dir string) *Storage {
		t.Helper()
		storage := NewStorage(dir, nil)
		vsix := createZip(t, map[string]string{
			"extension/package.json": `{"publisher":"golang","name":"go","version":"0.52.0"}`,
		})
		if err := storage.Install(context.Background(), domain.InstallParams{
			ID: id, Version: oldVersion, Data: vsix.Bytes(),
		}); err != nil {
			t.Fatalf("setup install: %v", err)
		}
		return storage
	}

	newVsix := func(t *testing.T) []byte {
		t.Helper()
		return createZip(t, map[string]string{
			"extension/package.json": `{"publisher":"golang","name":"go","version":"0.53.1"}`,
		}).Bytes()
	}

	t.Run("happy_path", func(t *testing.T) {
		dir := t.TempDir()
		storage := installOldVersion(t, dir)

		err := storage.Update(context.Background(), domain.InstallParams{
			ID: id, Version: newVersion, Data: newVsix(t),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Реестр содержит новую версию
		entries, err := readRegistry(filepath.Join(dir, registryFileName))
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Version != newVersion.String() {
			t.Errorf("registry version: got %s, want %s", entries[0].Version, newVersion.String())
		}

		// Новая директория существует
		newDir := filepath.Join(dir, "golang.go-0.53.1")
		if _, err := os.Stat(newDir); err != nil {
			t.Errorf("new version dir not found: %v", err)
		}

		// Старая директория удалена
		oldDir := filepath.Join(dir, "golang.go-0.52.0")
		if _, err := os.Stat(oldDir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("old version dir still exists")
		}
	})

	t.Run("not_installed", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewStorage(dir, nil)
		os.WriteFile(filepath.Join(dir, registryFileName), []byte("[]"), 0o644)

		err := storage.Update(context.Background(), domain.InstallParams{
			ID: id, Version: newVersion, Data: newVsix(t),
		})
		if !errors.Is(err, domain.ErrNotInstalled) {
			t.Errorf("got %v, want %v", err, domain.ErrNotInstalled)
		}
	})

	t.Run("same_version", func(t *testing.T) {
		dir := t.TempDir()
		storage := installOldVersion(t, dir)

		err := storage.Update(context.Background(), domain.InstallParams{
			ID: id, Version: oldVersion, Data: newVsix(t),
		})
		if !errors.Is(err, domain.ErrAlreadyInstalled) {
			t.Errorf("got %v, want %v", err, domain.ErrAlreadyInstalled)
		}
	})

	t.Run("invalid_zip_keeps_old_version", func(t *testing.T) {
		dir := t.TempDir()
		storage := installOldVersion(t, dir)

		err := storage.Update(context.Background(), domain.InstallParams{
			ID: id, Version: newVersion, Data: []byte("not a zip"),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Старая версия осталась в реестре
		entries, err := readRegistry(filepath.Join(dir, registryFileName))
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Version != oldVersion.String() {
			t.Errorf("registry version: got %s, want %s", entries[0].Version, oldVersion.String())
		}

		// Старая директория на месте
		oldDir := filepath.Join(dir, "golang.go-0.52.0")
		if _, err := os.Stat(oldDir); err != nil {
			t.Errorf("old version dir should still exist: %v", err)
		}
	})
}
