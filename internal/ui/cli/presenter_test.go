package cli

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

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
			presenter := NewPresenter(&buf, time.Millisecond, NewPacmanProgressBar(20))
			presenter.ShowExtensions(testCase.extensions)
			got := buf.String()

			if testCase.wantResults != got {
				t.Errorf("got %+v, want %+v", got, testCase.wantResults)
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
			presenter := NewPresenter(&buf, time.Millisecond, NewPacmanProgressBar(20))
			presenter.ShowMessage(testCase.msg)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestShowError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "simple_error",
			err:  errors.New("connection refused"),
			want: "connection refused\n",
		},
		{
			name: "wrapped_error",
			err:  fmt.Errorf("download: %w", errors.New("timeout")),
			want: "download: timeout\n",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			presenter := NewPresenter(&buf, time.Millisecond, NewPacmanProgressBar(20))
			presenter.ShowError(testCase.err)
			got := buf.String()

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
