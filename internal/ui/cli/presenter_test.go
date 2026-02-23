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
					Name:        "Go",
					Description: "Go support",
					Publisher: domain.Publisher{
						Name: "golang",
					},
				},
				{
					Name:        "Python",
					Description: "Python support",
					Publisher: domain.Publisher{
						Name: "microsoft team",
					},
				},
			},
			wantResults: "1. Go - Go support\n2. Python - Python support\n",
		},
		{
			name: "single_result",
			extensions: []domain.Extension{
				{
					Name:        "Python",
					Description: "Python support",
					Publisher: domain.Publisher{
						Name: "microsoft team",
					},
				},
			},
			wantResults: "1. Python - Python support\n",
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
