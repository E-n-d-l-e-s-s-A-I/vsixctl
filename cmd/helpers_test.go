package cmd

import (
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestParseInstallTargets(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []domain.InstallTarget
		wantErr bool
	}{
		{
			name: "without_version",
			args: []string{"ms-python.python"},
			want: []domain.InstallTarget{
				{ID: domain.ExtensionID{Publisher: "ms-python", Name: "python"}},
			},
		},
		{
			name: "with_version",
			args: []string{"ms-python.python@2024.8.1"},
			want: []domain.InstallTarget{
				{
					ID:      domain.ExtensionID{Publisher: "ms-python", Name: "python"},
					Version: &domain.Version{Major: 2024, Minor: 8, Patch: 1},
				},
			},
		},
		{
			name: "with_two_segment_version",
			args: []string{"golang.go@1.20"},
			want: []domain.InstallTarget{
				{
					ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
					Version: &domain.Version{Major: 1, Minor: 20, Patch: 0},
				},
			},
		},
		{
			name: "mixed_args",
			args: []string{"golang.go@1.0.0", "ms-python.python"},
			want: []domain.InstallTarget{
				{
					ID:      domain.ExtensionID{Publisher: "golang", Name: "go"},
					Version: &domain.Version{Major: 1, Minor: 0, Patch: 0},
				},
				{
					ID: domain.ExtensionID{Publisher: "ms-python", Name: "python"},
				},
			},
		},
		{
			name:    "multiple_at_signs",
			args:    []string{"ms-python.python@1@2"},
			wantErr: true,
		},
		{
			name:    "invalid_version",
			args:    []string{"ms-python.python@invalid"},
			wantErr: true,
		},
		{
			name:    "invalid_id",
			args:    []string{"@1.2.3"},
			wantErr: true,
		},
		{
			name:    "invalid_id_format",
			args:    []string{"invalid-format@1.0.0"},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseInstallTargets(testCase.args)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}
