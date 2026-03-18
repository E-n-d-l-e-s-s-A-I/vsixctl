package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

func TestVersions(t *testing.T) {
	goID := domain.ExtensionID{Publisher: "golang", Name: "go"}

	tests := []struct {
		name     string
		id       domain.ExtensionID
		limit    int
		registry *testutil.MockRegistry
		want     []domain.VersionInfo
		wantErr  string
	}{
		{
			name:  "returns_versions",
			id:    goID,
			limit: 10,
			registry: &testutil.MockRegistry{
				GetVersionsFunc: func(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
					return []domain.VersionInfo{
						{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
						{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: false},
					}, nil
				},
			},
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: false},
			},
		},
		{
			name:  "empty_versions",
			id:    goID,
			limit: 10,
			registry: &testutil.MockRegistry{
				GetVersionsFunc: func(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
					return nil, nil
				},
			},
			want: nil,
		},
		{
			name:  "registry_error",
			id:    goID,
			limit: 10,
			registry: &testutil.MockRegistry{
				GetVersionsFunc: func(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
					return nil, errors.New("connection refused")
				},
			},
			wantErr: "connection refused",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			svc := NewUseCaseService(testCase.registry, nil, nil, 1)

			got, err := svc.Versions(t.Context(), testCase.id, testCase.limit)

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
