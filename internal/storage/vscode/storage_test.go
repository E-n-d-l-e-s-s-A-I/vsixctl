package vscode

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestParseExtensionDir(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string // содержимое package.json ("" — не создавать файл)
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

// writePackageJSON — хелпер для создания директории расширения с package.json
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
