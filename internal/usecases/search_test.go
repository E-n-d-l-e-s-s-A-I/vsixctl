package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

func TestSearch(t *testing.T) {
	goExt := domain.Extension{
		ID:          domain.ExtensionID{Publisher: "golang", Name: "go"},
		Description: "Go support",
		Version:     domain.Version{Major: 0, Minor: 53, Patch: 1},
	}
	pythonExt := domain.Extension{
		ID:          domain.ExtensionID{Publisher: "ms-python", Name: "python"},
		Description: "Python support",
		Version:     domain.Version{Major: 2026, Minor: 2, Patch: 0},
	}

	tests := []struct {
		name     string
		query    string
		count    int
		registry *testutil.MockRegistry
		want     []domain.Extension
		wantErr  string
	}{
		{
			name:  "multiple_results",
			query: "go",
			count: 10,
			registry: &testutil.MockRegistry{
				SearchFunc: func(ctx context.Context, query string, count int) ([]domain.Extension, error) {
					return []domain.Extension{goExt, pythonExt}, nil
				},
			},
			want: []domain.Extension{goExt, pythonExt},
		},
		{
			name:  "empty_results",
			query: "nonexistent",
			count: 10,
			registry: &testutil.MockRegistry{
				SearchFunc: func(ctx context.Context, query string, count int) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
			},
			want: []domain.Extension{},
		},
		{
			name:  "registry_error",
			query: "go",
			count: 10,
			registry: &testutil.MockRegistry{
				SearchFunc: func(ctx context.Context, query string, count int) ([]domain.Extension, error) {
					return nil, errors.New("connection refused")
				},
			},
			wantErr: "connection refused",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			svc := NewUseCaseService(testCase.registry, nil, nil, 1)

			got, err := svc.Search(t.Context(), testCase.query, testCase.count)

			if testCase.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", testCase.wantErr)
				}
				if err.Error() != testCase.wantErr {
					t.Fatalf("got error %q, want %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}
