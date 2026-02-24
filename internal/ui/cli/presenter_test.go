package cli

import (
	"bytes"
	"testing"

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
			presenter := NewPresenter(&buf)
			presenter.ShowExtensions(testCase.extensions)
			got := buf.String()

			if testCase.wantResults != got {
				t.Errorf("got %+v, want %+v", got, testCase.wantResults)
			}
		})
	}
}
