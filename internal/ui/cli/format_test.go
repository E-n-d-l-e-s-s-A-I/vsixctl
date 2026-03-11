package cli

import (
	"fmt"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestFormatExtension(t *testing.T) {
	tests := []struct {
		name  string
		index int
		ext   domain.Extension
		want  string
	}{
		{
			name:  "basic_extension",
			index: 1,
			ext: domain.Extension{
				ID:          domain.ExtensionID{Publisher: "ms-python", Name: "python"},
				Description: "Python language support",
			},
			want: "1. ms-python.python - Python language support",
		},
		{
			name:  "empty_description",
			index: 5,
			ext: domain.Extension{
				ID: domain.ExtensionID{Publisher: "pub", Name: "ext"},
			},
			want: "5. pub.ext - ",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatExtension(testCase.index, testCase.ext)
			if got != testCase.want {
				t.Errorf("FormatExtension() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "direct_sentinel",
			err:  domain.ErrNotFound,
			want: "extension not found",
		},
		{
			name: "wrapped_once",
			err:  fmt.Errorf("get extension: %w", domain.ErrNotFound),
			want: "extension not found",
		},
		{
			name: "wrapped_twice",
			err:  fmt.Errorf("get latest version: %w", fmt.Errorf("get extension: %w", domain.ErrNotFound)),
			want: "extension not found",
		},
		{
			name: "already_installed",
			err:  domain.ErrAlreadyInstalled,
			want: "extension already installed",
		},
		{
			name: "version_not_found",
			err:  fmt.Errorf("get latest version: %w", domain.ErrVersionNotFound),
			want: "compatible version not found",
		},
		{
			name: "all_sources_unavailable",
			err:  fmt.Errorf("download: %w", domain.ErrAllSourcesUnavailable),
			want: "download failed: all sources unavailable",
		},
		{
			name: "unknown_error_fallback",
			err:  fmt.Errorf("unexpected status code 500"),
			want: "unexpected status code 500",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatError(testCase.err)
			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFormatInstallPlan(t *testing.T) {
	tests := []struct {
		name         string
		requestedIDs []domain.ExtensionID
		extensions   []domain.DownloadInfo
		want         string
	}{
		{
			name: "only_requested",
			requestedIDs: []domain.ExtensionID{
				{Publisher: "ms-python", Name: "python"},
			},
			extensions: []domain.DownloadInfo{
				{
					ID:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
					Version: domain.Version{Major: 2024, Minor: 1, Patch: 0},
					Size:    5 * 1024 * 1024, // 5 MiB
				},
			},
			want: "\nExtensions (1):\n  ms-python.python-2024.1.0  5.0 MiB\n\nTotal Size: 5.0 MiB",
		},
		{
			name: "with_dependencies",
			requestedIDs: []domain.ExtensionID{
				{Publisher: "ms-python", Name: "python"},
			},
			extensions: []domain.DownloadInfo{
				{
					ID:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
					Version: domain.Version{Major: 2024, Minor: 1, Patch: 0},
					Size:    5 * 1024 * 1024,
				},
				{
					ID:      domain.ExtensionID{Publisher: "ms-python", Name: "vscode-pylance"},
					Version: domain.Version{Major: 2024, Minor: 2, Patch: 3},
					Size:    3 * 1024 * 1024,
				},
			},
			want: "\nExtensions (1):\n  ms-python.python-2024.1.0  5.0 MiB\n\nDependencies (1):\n  ms-python.vscode-pylance-2024.2.3  3.0 MiB\n\nTotal Size: 8.0 MiB",
		},
		{
			name: "only_dependencies",
			requestedIDs: []domain.ExtensionID{
				{Publisher: "removed", Name: "ext"},
			},
			extensions: []domain.DownloadInfo{
				{
					ID:      domain.ExtensionID{Publisher: "dep", Name: "one"},
					Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
					Size:    1024,
				},
			},
			want: "\nDependencies (1):\n  dep.one-1.0.0  1.0 KiB\n\nTotal Size: 1.0 KiB",
		},
		{
			name: "multiple_sorted_alphabetically",
			requestedIDs: []domain.ExtensionID{
				{Publisher: "z-pub", Name: "ext"},
				{Publisher: "a-pub", Name: "ext"},
			},
			extensions: []domain.DownloadInfo{
				{
					ID:      domain.ExtensionID{Publisher: "z-pub", Name: "ext"},
					Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
					Size:    512,
				},
				{
					ID:      domain.ExtensionID{Publisher: "a-pub", Name: "ext"},
					Version: domain.Version{Major: 2, Minor: 0, Patch: 0},
					Size:    256,
				},
			},
			want: "\nExtensions (2):\n  a-pub.ext-2.0.0  256 B\n  z-pub.ext-1.0.0  512 B\n\nTotal Size: 768 B",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatInstallPlan(testCase.requestedIDs, testCase.extensions)
			if got != testCase.want {
				t.Errorf("FormatInstallPlan()\ngot:  %q\nwant: %q", got, testCase.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero_bytes",
			bytes: 0,
			want:  "0 B",
		},
		{
			name:  "small_bytes",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "exactly_1_kib",
			bytes: 1024,
			want:  "1.0 KiB",
		},
		{
			name:  "fractional_kib",
			bytes: 1536, // 1.5 KiB
			want:  "1.5 KiB",
		},
		{
			name:  "exactly_1_mib",
			bytes: 1024 * 1024,
			want:  "1.0 MiB",
		},
		{
			name:  "fractional_mib",
			bytes: 5 * 1024 * 1024, // 5.0 MiB
			want:  "5.0 MiB",
		},
		{
			name:  "exactly_1_gib",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GiB",
		},
		{
			name:  "fractional_gib",
			bytes: 1536 * 1024 * 1024, // 1.5 GiB
			want:  "1.5 GiB",
		},
		{
			name:  "just_below_kib",
			bytes: 1023,
			want:  "1023 B",
		},
		{
			name:  "just_below_mib",
			bytes: 1024*1024 - 1,
			want:  "1024.0 KiB",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatSize(testCase.bytes)
			if got != testCase.want {
				t.Errorf("FormatSize(%d) = %q, want %q", testCase.bytes, got, testCase.want)
			}
		})
	}
}

func TestFormatUpdatePlan(t *testing.T) {
	tests := []struct {
		name     string
		toUpdate []domain.UpdateInfo
		want     string
	}{
		{
			name: "single_update",
			toUpdate: []domain.UpdateInfo{
				{
					Prev: domain.Extension{
						ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
						Version: domain.Version{Major: 0, Minor: 41, Patch: 0},
					},
					New: domain.DownloadInfo{
						ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
						Version: domain.Version{Major: 0, Minor: 42, Patch: 0},
						Size:    15 * 1024 * 1024,
					},
				},
			},
			want: "\nUpdates (1):\n  golang.go  0.41.0 -> 0.42.0  15.0 MiB\n\nTotal Download Size: 15.0 MiB",
		},
		{
			name: "multiple_sorted_alphabetically",
			toUpdate: []domain.UpdateInfo{
				{
					Prev: domain.Extension{
						ID:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
						Version: domain.Version{Major: 2024, Minor: 1, Patch: 0},
					},
					New: domain.DownloadInfo{
						ID:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
						Version: domain.Version{Major: 2024, Minor: 2, Patch: 0},
						Size:    22 * 1024 * 1024,
					},
				},
				{
					Prev: domain.Extension{
						ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
						Version: domain.Version{Major: 0, Minor: 41, Patch: 0},
					},
					New: domain.DownloadInfo{
						ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
						Version: domain.Version{Major: 0, Minor: 42, Patch: 0},
						Size:    15 * 1024 * 1024,
					},
				},
			},
			want: "\nUpdates (2):\n  golang.go  0.41.0 -> 0.42.0  15.0 MiB\n  ms-python.python  2024.1.0 -> 2024.2.0  22.0 MiB\n\nTotal Download Size: 37.0 MiB",
		},
		{
			name: "patch_update",
			toUpdate: []domain.UpdateInfo{
				{
					Prev: domain.Extension{
						ID:      domain.ExtensionID{Publisher: "pub", Name: "ext"},
						Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
					},
					New: domain.DownloadInfo{
						ID:      domain.ExtensionID{Publisher: "pub", Name: "ext"},
						Version: domain.Version{Major: 1, Minor: 0, Patch: 1},
						Size:    512,
					},
				},
			},
			want: "\nUpdates (1):\n  pub.ext  1.0.0 -> 1.0.1  512 B\n\nTotal Download Size: 512 B",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatUpdatePlan(testCase.toUpdate)
			if got != testCase.want {
				t.Errorf("FormatUpdatePlan()\ngot:  %q\nwant: %q", got, testCase.want)
			}
		})
	}
}

func TestFormatInstallResult(t *testing.T) {
	tests := []struct {
		name   string
		result domain.ExtensionResult
		want   string
	}{
		{
			name:   "successful",
			result: domain.ExtensionResult{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}},
			want:   "golang.go: installed",
		},
		{
			name:   "direct_error",
			result: domain.ExtensionResult{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrAlreadyInstalled},
			want:   "golang.go: extension already installed",
		},
		{
			name:   "wrapped_error",
			result: domain.ExtensionResult{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("get latest version: %w", domain.ErrNotFound)},
			want:   "unknown.ext: extension not found",
		},
		{
			name:   "unknown_error",
			result: domain.ExtensionResult{ID: domain.ExtensionID{Publisher: "broken", Name: "pkg"}, Err: fmt.Errorf("network timeout")},
			want:   "broken.pkg: network timeout",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := formatInstallResult(testCase.result)
			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
