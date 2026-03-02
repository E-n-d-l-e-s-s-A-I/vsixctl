package vscode

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

			storage := NewVSCodeStorage(storagePath)
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
			name: "path_traversal",
			zipFiles: map[string]string{
				"extension/../../etc/passwd": "root:x:0:0",
				"extension/safe.js":          "ok",
			},
			wantFiles: []string{
				"safe.js",
			},
		},
	}

	id := domain.ExtensionID{Publisher: "golang", Name: "go"}
	version := domain.Version{Major: 1, Minor: 0, Patch: 0}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			storage := NewVSCodeStorage(dir)

			vsix := createZip(t, testCase.zipFiles)
			err := storage.Install(context.Background(), id, version, vsix)

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
	storage := NewVSCodeStorage(dir)

	id := domain.ExtensionID{Publisher: "test", Name: "ext"}
	version := domain.Version{Major: 1, Minor: 0, Patch: 0}

	err := storage.Install(context.Background(), id, version, strings.NewReader("not a zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

// createZip - хелпер для создания zip-архива в памяти
func createZip(t *testing.T, files map[string]string) *bytes.Reader {
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
	return bytes.NewReader(buf.Bytes())
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
