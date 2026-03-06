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
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: fmt.Errorf("extension already installed")},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("extension not found")},
			},
			want: "golang.go: extension already installed\nunknown.ext: extension not found\n",
		},
		{
			name: "mixed_results",
			results: []domain.InstallResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: nil},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("extension not found")},
			},
			want: "golang.go: installed\nunknown.ext: extension not found\n",
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
