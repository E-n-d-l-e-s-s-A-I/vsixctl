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
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
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
		results []domain.ExtensionResult
		want    string
	}{
		{
			name: "all_successful",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: nil},
				{ID: domain.ExtensionID{Publisher: "ms-python", Name: "python"}, Err: nil},
			},
			want: "golang.go: installed\nms-python.python: installed\n",
		},
		{
			name: "all_failed",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrAlreadyInstalled},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: domain.ErrNotFound},
			},
			want: "golang.go: extension already installed\nunknown.ext: extension not found\n",
		},
		{
			name: "mixed_results",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: nil},
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: domain.ErrNotFound},
			},
			want: "golang.go: installed\nunknown.ext: extension not found\n",
		},
		{
			name: "wrapped_errors",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "unknown", Name: "ext"}, Err: fmt.Errorf("get latest version: %w", domain.ErrNotFound)},
				{ID: domain.ExtensionID{Publisher: "broken", Name: "pkg"}, Err: fmt.Errorf("get latest version: %w", domain.ErrVersionNotFound)},
			},
			want: "unknown.ext: extension not found\nbroken.pkg: compatible version not found\n",
		},
		{
			name:    "empty_results",
			results: []domain.ExtensionResult{},
			want:    "",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
			presenter.ShowInstallResult(testCase.results)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestShowRemoveResult(t *testing.T) {
	tests := []struct {
		name    string
		results []domain.ExtensionResult
		want    string
	}{
		{
			name: "successful_remove",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}},
			},
			want: "golang.go: deleted\n",
		},
		{
			name: "not_installed",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrNotInstalled},
			},
			want: "golang.go: extension not installed\n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
			presenter.ShowRemoveResult(testCase.results)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestShowUpdateResult(t *testing.T) {
	tests := []struct {
		name    string
		results []domain.ExtensionResult
		want    string
	}{
		{
			name: "successful_update",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}},
			},
			want: "golang.go: updated\n",
		},
		{
			name: "not_installed",
			results: []domain.ExtensionResult{
				{ID: domain.ExtensionID{Publisher: "golang", Name: "go"}, Err: domain.ErrNotInstalled},
			},
			want: "golang.go: extension not installed\n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
			presenter.ShowUpdateResult(testCase.results)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		msg     string
		want    string
	}{
		{
			name:    "verbose_enabled",
			verbose: true,
			msg:     "source unavailable",
			want:    "source unavailable\n",
		},
		{
			name:    "verbose_disabled",
			verbose: false,
			msg:     "source unavailable",
			want:    "",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), testCase.verbose)
			presenter.Log(testCase.msg)
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
			presenter := NewPresenter(&buf, nil, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
			presenter.ShowMessage(testCase.msg)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name      string
		userInput string
		want      bool
	}{
		{
			name:      "user_agree",
			userInput: "y",
			want:      true,
		},
		{
			name:      "user_agree_upper_case",
			userInput: "Y",
			want:      true,
		},
		{
			name:      "user_agree_empty_input",
			userInput: "",
			want:      true,
		},
		{
			name:      "user_disagree",
			userInput: "n",
			want:      false,
		},
		{
			name:      "user_disagree_random_letter",
			userInput: "h",
			want:      false,
		},
	}

	// Все Confirm* методы используют одну и ту же логику confirm(),
	// проверяем через ConfirmInstall как представитель
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var in bytes.Buffer
			var out bytes.Buffer

			presenter := NewPresenter(&out, &in, func() int { return TerminalWidth }, time.Millisecond, cliutils.NewPacmanProgressBar(), false)
			fmt.Fprintln(&in, testCase.userInput)

			got := presenter.ConfirmInstall(nil, nil)
			if got != testCase.want {
				t.Errorf("got %t, want %t", got, testCase.want)
			}
		})
	}
}
