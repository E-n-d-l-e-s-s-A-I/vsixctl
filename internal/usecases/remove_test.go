package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

// confirmRemoveOpts - RemoveOpts, которые всегда подтверждают удаление
func confirmRemoveOpts() RemoveOpts {
	return RemoveOpts{
		Confirm: func([]domain.ExtensionID, []domain.Extension) bool { return true },
	}
}

func rejectRemoveOpts() RemoveOpts {
	return RemoveOpts{
		Confirm: func([]domain.ExtensionID, []domain.Extension) bool { return false },
	}
}

func TestRemove(t *testing.T) {
	goID := domain.ExtensionID{Publisher: "golang", Name: "go"}
	pythonID := domain.ExtensionID{Publisher: "ms-python", Name: "python"}
	depID := domain.ExtensionID{Publisher: "some", Name: "dep"}
	subDepID := domain.ExtensionID{Publisher: "some", Name: "subdep"}

	goExt := domain.Extension{ID: goID}
	pythonExt := domain.Extension{ID: pythonID}
	depExt := domain.Extension{ID: depID}
	subDepExt := domain.Extension{ID: subDepID}

	goExtWithPack := domain.Extension{
		ID:            goID,
		ExtensionPack: []domain.ExtensionID{depID},
	}
	depExtWithSubPack := domain.Extension{
		ID:            depID,
		ExtensionPack: []domain.ExtensionID{subDepID},
	}

	diskFull := errors.New("disk full")

	tests := []struct {
		name    string
		ids     []domain.ExtensionID
		opts    RemoveOpts
		storage *testutil.MockStorage
		want    []domain.ExtensionResult
		wantErr error
	}{
		{
			name: "single_extension",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}},
		},
		{
			name: "not_installed",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts: RemoveOpts{},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: domain.ErrNotInstalled}},
		},
		{
			name: "mixed_installed_and_not",
			ids:  []domain.ExtensionID{goID, pythonID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{
				{ID: pythonID, Err: domain.ErrNotInstalled},
				{ID: goID},
			},
		},
		{
			name: "confirm_rejected",
			ids:  []domain.ExtensionID{goID},
			opts: rejectRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt}, nil
				},
			},
			want: nil,
		},
		{
			name: "extension_pack",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExtWithPack, depExt}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}},
		},
		{
			name: "nested_extension_pack",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExtWithPack, depExtWithSubPack, subDepExt}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}, {ID: subDepID}},
		},
		{
			name: "pack_member_not_installed",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExtWithPack}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}},
		},
		{
			// goExt ссылается на depExt, depExt ссылается обратно на goExt - цикл
			name: "circular_extension_pack",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{
						{ID: goID, ExtensionPack: []domain.ExtensionID{depID}},
						{ID: depID, ExtensionPack: []domain.ExtensionID{goID}},
					}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}},
		},
		{
			name: "shared_pack_member",
			ids:  []domain.ExtensionID{goID, pythonID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{
						{ID: goID, ExtensionPack: []domain.ExtensionID{depID}},
						{ID: pythonID, ExtensionPack: []domain.ExtensionID{depID}},
						depExt,
					}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}, {ID: pythonID}},
		},
		{
			name: "storage_remove_error",
			ids:  []domain.ExtensionID{goID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt}, nil
				},
				RemoveFunc: func(_ context.Context, _ domain.ExtensionID) error {
					return diskFull
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: diskFull}},
		},
		{
			name: "storage_remove_error_with_one_ext",
			ids:  []domain.ExtensionID{goID, pythonID},
			opts: confirmRemoveOpts(),
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goExt, pythonExt}, nil
				},
				RemoveFunc: func(_ context.Context, id domain.ExtensionID) error {
					if id == pythonID {
						return diskFull
					}
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID, Err: diskFull}},
		},
		{
			name: "storage_list_error",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts: RemoveOpts{},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return nil, domain.ErrExtensionDirNotFound
				},
			},
			wantErr: domain.ErrExtensionDirNotFound,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			svc := NewUseCaseService(nil, testCase.storage, nil, 1)

			report, err := svc.Remove(t.Context(), testCase.ids, testCase.opts)

			if testCase.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", testCase.wantErr)
				}
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("got error %v, want %v", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertResults(t, report.Results, testCase.want)
		})
	}
}
