package vscode

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

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
			name:        "invalid_version",
			packageJSON: `{"publisher":"test","name":"ext","version":"abc"}`,
			wantErr:     true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()

			if testCase.packageJSON != "" {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte(testCase.packageJSON), 0o644)
			}

			got, err := ParseExtensionDir(dir)

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
		name    string
		setup   func(dir string)
		want    []domain.Extension
		wantErr bool
	}{
		{
			name: "multiple_extensions",
			setup: func(dir string) {
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
				writePackageJSON(t, dir, "ms-python.python-2026.2.0",
					`{"publisher":"ms-python","name":"python","version":"2026.2.0","description":"Python support"}`)
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
			name:  "empty_directory",
			setup: func(dir string) {},
			want:  []domain.Extension{},
		},
		{
			name: "skips_files",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "extensions.json"), []byte("{}"), 0o644)
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
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
			name: "skips_broken_extensions",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "broken-ext"), 0o755)
				writePackageJSON(t, dir, "golang.go-0.53.1",
					`{"publisher":"golang","name":"go","version":"0.53.1","description":"Go support"}`)
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
			name:    "directory_not_exists",
			setup:   nil,
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(dir)
			}

			storagePath := dir
			if testCase.wantErr {
				storagePath = filepath.Join(dir, "nonexistent")
			}

			storage := NewVSCodeStorage(storagePath, nil)
			got, err := storage.List(context.Background())

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
	versionInfo := domain.VersionInfo{Version: version}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			storage := NewVSCodeStorage(dir, nil)

			vsix := createZip(t, testCase.zipFiles)
			err := storage.Install(context.Background(), id, versionInfo, vsix.Bytes())

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

				// .vsixmanifest извлекается из корня архива — проверяем содержимое
				if wantFile == ".vsixmanifest" {
					want := testCase.zipFiles["extension.vsixmanifest"]
					if string(got) != want {
						t.Errorf("file %s: got %q, want %q", wantFile, string(got), want)
					}
					continue
				}

				// package.json модифицируется injectMetadata — проверяем содержимое __metadata
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
	storage := NewVSCodeStorage(dir, nil)

	id := domain.ExtensionID{Publisher: "test", Name: "ext"}
	version := domain.Version{Major: 1, Minor: 0, Patch: 0}
	versionInfo := domain.VersionInfo{Version: version}

	err := storage.Install(context.Background(), id, versionInfo, []byte("not a zip"))
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
		id               domain.ExtensionID
		version          domain.Version
		relativeLocation string
		wantCount        int
		wantErr          bool
	}{
		{
			name:             "no_existing_file",
			existingRegistry: "",
			id:               domain.ExtensionID{Publisher: "golang", Name: "go"},
			version:          domain.Version{Major: 0, Minor: 53, Patch: 1},
			relativeLocation: "golang.go-0.53.1",
			wantCount:        1,
		},
		{
			name:             "empty_registry",
			existingRegistry: "[]",
			id:               domain.ExtensionID{Publisher: "golang", Name: "go"},
			version:          domain.Version{Major: 0, Minor: 53, Patch: 1},
			relativeLocation: "golang.go-0.53.1",
			wantCount:        1,
		},
		{
			name:             "append_to_existing",
			existingRegistry: `[{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"/ext/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0"}]`,
			id:               domain.ExtensionID{Publisher: "golang", Name: "go"},
			version:          domain.Version{Major: 0, Minor: 53, Patch: 1},
			relativeLocation: "golang.go-0.53.1",
			wantCount:        2,
		},
		{
			name:             "update_existing",
			existingRegistry: `[{"identifier":{"id":"golang.go"},"version":"0.52.0","location":{"$mid":1,"path":"/ext/golang.go-0.52.0","scheme":"file"},"relativeLocation":"golang.go-0.52.0"}]`,
			id:               domain.ExtensionID{Publisher: "golang", Name: "go"},
			version:          domain.Version{Major: 0, Minor: 53, Patch: 1},
			relativeLocation: "golang.go-0.53.1",
			wantCount:        1,
		},
		{
			name:             "invalid_json",
			existingRegistry: "{not json}",
			id:               domain.ExtensionID{Publisher: "golang", Name: "go"},
			version:          domain.Version{Major: 0, Minor: 53, Patch: 1},
			relativeLocation: "golang.go-0.53.1",
			wantErr:          true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			storage := NewVSCodeStorage(dir, nil)

			if testCase.existingRegistry != "" {
				os.WriteFile(filepath.Join(dir, registryFileName), []byte(testCase.existingRegistry), 0o644)
			}

			err := storage.registerExtension(testCase.id, testCase.version, testCase.relativeLocation)

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
				if e.Identifier.ID == testCase.id.String() {
					found = &entries[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("entry %s not found in registry", testCase.id.String())
			}
			if found.Version != testCase.version.String() {
				t.Errorf("version: got %s, want %s", found.Version, testCase.version.String())
			}
			if found.RelativeLocation != testCase.relativeLocation {
				t.Errorf("relativeLocation: got %s, want %s", found.RelativeLocation, testCase.relativeLocation)
			}
			wantPath := filepath.Join(dir, testCase.relativeLocation)
			if found.Location.Path != wantPath {
				t.Errorf("location.path: got %s, want %s", found.Location.Path, wantPath)
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
	storage := NewVSCodeStorage(dir, nil)

	existingJSON := `[{"identifier":{"id":"ms-python.python"},"version":"2026.2.0","location":{"$mid":1,"path":"/ext/ms-python.python-2026.2.0","scheme":"file"},"relativeLocation":"ms-python.python-2026.2.0","metadata":{"publisherDisplayName":"Microsoft","installedTimestamp":1770717444996}}]`
	os.WriteFile(filepath.Join(dir, registryFileName), []byte(existingJSON), 0o644)

	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	ver := domain.Version{Major: 0, Minor: 53, Patch: 1}
	if err := storage.registerExtension(id, ver, "golang.go-0.53.1"); err != nil {
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
	storage := NewVSCodeStorage(dir, nil)

	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	versionInfo := domain.VersionInfo{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}}

	vsix := createZip(t, map[string]string{
		"extension/package.json": `{"name":"go"}`,
	})
	err := storage.Install(context.Background(), id, versionInfo, vsix.Bytes())
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
}
