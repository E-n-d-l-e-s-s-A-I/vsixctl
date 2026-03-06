package cli

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
)

const TerminalWidth = 80

func TestShowExtensions(t *testing.T) {
	tests := []struct {
		name        string
		extensions  []domain.Extension
		wantResults string
	}{
		{
			name: "many_results",
			extensions: []domain.Extension{
				{
					ID: domain.ExtensionID{
						Name:      "go",
						Publisher: "golang",
					},
					Description: "Go support",
				},
				{
					ID: domain.ExtensionID{
						Name:      "python",
						Publisher: "ms-python",
					},
					Description: "Python support",
				},
			},
			wantResults: "1. golang.go - Go support\n2. ms-python.python - Python support\n",
		},
		{
			name: "single_result",
			extensions: []domain.Extension{
				{
					ID: domain.ExtensionID{
						Name:      "python",
						Publisher: "ms-python",
					},
					Description: "Python support",
				},
			},
			wantResults: "1. ms-python.python - Python support\n",
		},
		{
			name:        "empty_result",
			extensions:  []domain.Extension{},
			wantResults: "no results\n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar())
			presenter.ShowExtensions(testCase.extensions)
			got := buf.String()

			if testCase.wantResults != got {
				t.Errorf("got %+v, want %+v", got, testCase.wantResults)
			}
		})
	}
}

func TestShowInstallResult(t *testing.T) {
	tests := []struct {
		name    string
		results []domain.InstallResult
		want    string
	}{
		{
			name: "all_successful",
			results: []domain.InstallResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: nil},
				{ID: domain.ExtensionID{Publisher: "ms-python", Name: "python"}, Err: nil},
			},
			want: "golang.go: installed\nms-python.python: installed\n",
		},
		{
			name: "all_failed",
			results: []domain.InstallResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrAlreadyInstalled},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: domain.ErrNotFound},
			},
			want: "golang.go: extension already installed\nunknown.ext: extension not found\n",
		},
		{
			name: "mixed_results",
			results: []domain.InstallResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: nil},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: domain.ErrNotFound},
			},
			want: "golang.go: installed\nunknown.ext: extension not found\n",
		},
		{
			name: "wrapped_errors",
			results: []domain.InstallResult{
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("get latest version: %w", domain.ErrNotFound)},
				{ID: domain.ExtensionID{Publisher: "broken", Name: "pkg"}, Err: fmt.Errorf("get latest version: %w", domain.ErrVersionNotFound)},
			},
			want: "unknown.ext: extension not found\nbroken.pkg: compatible version not found\n",
		},
		{
			name:    "empty_results",
			results: []domain.InstallResult{},
			want:    "",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar())
			presenter.ShowInstallResult(testCase.results)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
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
			got := FormatError(testCase.err)
			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFormatInstallResult(t *testing.T) {
	tests := []struct {
		name   string
		result domain.InstallResult
		want   string
	}{
		{
			name:   "successful",
			result: domain.InstallResult{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}},
			want:   "golang.go: installed",
		},
		{
			name:   "direct_error",
			result: domain.InstallResult{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrAlreadyInstalled},
			want:   "golang.go: extension already installed",
		},
		{
			name:   "wrapped_error",
			result: domain.InstallResult{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("get latest version: %w", domain.ErrNotFound)},
			want:   "unknown.ext: extension not found",
		},
		{
			name:   "unknown_error",
			result: domain.InstallResult{ID: domain.ExtensionID{Publisher: "broken", Name: "pkg"}, Err: fmt.Errorf("network timeout")},
			want:   "broken.pkg: network timeout",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := FormatInstallResult(testCase.result)
			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestShowMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "simple_message",
			msg:  "installation complete",
			want: "installation complete\n",
		},
		{
			name: "empty_message",
			msg:  "",
			want: "\n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar())
			presenter.ShowMessage(testCase.msg)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
