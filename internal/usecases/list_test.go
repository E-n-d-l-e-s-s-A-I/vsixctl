package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

func TestList(t *testing.T) {
	goExt := domain.Extension{
		ID:          domain.ExtensionID{Publisher: "golang", Name: "go"},
		Description: "Go support",
		Version:     domain.Version{Major: 0, Minor: 53, Patch: 1},
	}
	pythonExt := domain.Extension{
		ID:          domain.ExtensionID{Publisher: "ms-python", Name: "python"},
		Description: "Python support",
		Version:     domain.Version{Major: 2026, Minor: 2, Patch: 0},
		Platform:    domain.LinuxX64,
	}

	tests := []struct {
		name    string
		storage *testutil.MockStorage
		want    []domain.Extension
		wantErr error
	}{
		{
			name: "multiple_extensions",
			storage: &testutil.MockStorage{
				ListFunc: func(ctx context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt, pythonExt}, nil
				},
			},
			want: []domain.Extension{goExt, pythonExt},
		},
		{
			name: "empty_list",
			storage: &testutil.MockStorage{
				ListFunc: func(ctx context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
			},
			want: []domain.Extension{},
		},
		{
			name: "storage_error",
			storage: &testutil.MockStorage{
				ListFunc: func(ctx context.Context) ([]domain.Extension, error) {
					return nil, domain.ErrExtensionDirNotFound
				},
			},
			wantErr: domain.ErrExtensionDirNotFound,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			svc := NewUseCaseService(nil, testCase.storage, nil, 1)

			got, err := svc.List(t.Context())

			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("got error %v, want %v", err, testCase.wantErr)
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
