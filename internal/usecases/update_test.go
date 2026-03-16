package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

// noopUpdateOpts - UpdateOpts, которые всегда подтверждают и не отслеживают прогресс
func noopUpdateOpts() UpdateOpts {
	return UpdateOpts{
		Confirm:           func([]domain.UpdateInfo) bool { return true },
		OnProgressFactory: func(string) (domain.ProgressFunc, func()) { return func(int64, int64) {}, func() {} },
	}
}

func rejectUpdateOpts() UpdateOpts {
	opts := noopUpdateOpts()
	opts.Confirm = func([]domain.UpdateInfo) bool { return false }
	return opts
}

func TestUpdate(t *testing.T) {
	goID := domain.ExtensionID{Publisher: "golang", Name: "go"}
	pythonID := domain.ExtensionID{Publisher: "ms-python", Name: "python"}

	goOldVer := domain.Version{Major: 0, Minor: 53, Patch: 0}
	goNewVer := domain.Version{Major: 0, Minor: 54, Patch: 0}
	pyOldVer := domain.Version{Major: 2026, Minor: 2, Patch: 0}
	pyNewVer := domain.Version{Major: 2026, Minor: 3, Patch: 0}

	goInstalled := domain.Extension{ID: goID, Version: goOldVer, Platform: domain.LinuxX64}
	pythonInstalled := domain.Extension{ID: pythonID, Version: pyOldVer, Platform: domain.LinuxX64}

	goDownloadNew := domain.DownloadInfo{
		ID:       goID,
		Version:  goNewVer,
		Platform: domain.LinuxX64,
		Source:   "https://example.com/go.vsix",
	}
	pythonDownloadNew := domain.DownloadInfo{
		ID:       pythonID,
		Version:  pyNewVer,
		Platform: domain.LinuxX64,
		Source:   "https://example.com/python.vsix",
	}
	goDownloadSame := domain.DownloadInfo{
		ID:       goID,
		Version:  goOldVer,
		Platform: domain.LinuxX64,
		Source:   "https://example.com/go.vsix",
	}

	connRefused := errors.New("connection refused")
	diskFull := errors.New("disk full")

	tests := []struct {
		name     string
		ids      []domain.ExtensionID
		opts     UpdateOpts
		registry *testutil.MockRegistry
		storage  *testutil.MockStorage
		want     []domain.ExtensionResult
		wantErr  error
	}{
		{
			name: "single_extension",
			ids:  []domain.ExtensionID{goID},
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadNew, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}},
		},
		{
			name: "already_up_to_date",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts: UpdateOpts{},
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadSame, nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
			},
			want: nil,
		},
		{
			name: "not_installed",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts:     UpdateOpts{},
			registry: &testutil.MockRegistry{},
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
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadNew, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
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
			opts: rejectUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadNew, nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
			},
			want: nil,
		},
		{
			name: "update_all",
			ids:  nil,
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID}, goDownloadNew, nil
					case pythonID:
						return domain.Extension{ID: pythonID}, pythonDownloadNew, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled, pythonInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID}},
		},
		{
			name: "download_error",
			ids:  []domain.ExtensionID{goID},
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadNew, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return nil, connRefused
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: connRefused}},
		},
		{
			name: "download_error_with_one_ext",
			ids:  []domain.ExtensionID{goID, pythonID},
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID}, goDownloadNew, nil
					case pythonID:
						return domain.Extension{ID: pythonID}, pythonDownloadNew, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, info domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					if info.ID == pythonID {
						return nil, connRefused
					}
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled, pythonInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID, Err: connRefused}},
		},
		{
			name: "storage_update_error",
			ids:  []domain.ExtensionID{goID},
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{ID: goID}, goDownloadNew, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return diskFull
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: diskFull}},
		},
		{
			name: "storage_update_error_with_one_ext",
			ids:  []domain.ExtensionID{goID, pythonID},
			opts: noopUpdateOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID}, goDownloadNew, nil
					case pythonID:
						return domain.Extension{ID: pythonID}, pythonDownloadNew, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled, pythonInstalled}, nil
				},
				UpdateFunc: func(_ context.Context, id domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					if id == pythonID {
						return diskFull
					}
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID, Err: diskFull}},
		},
		{
			name: "resolve_error",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts: UpdateOpts{},
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{}, domain.DownloadInfo{}, connRefused
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{goInstalled}, nil
				},
			},
			wantErr: connRefused,
		},
		{
			name: "storage_list_error",
			ids:  []domain.ExtensionID{goID},
			// Confirm не задан - паника если будет вызван
			opts: UpdateOpts{},
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
			svc := NewUseCaseService(testCase.registry, testCase.storage, nil, 1)

			report, err := svc.Update(t.Context(), testCase.ids, testCase.opts)

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

	t.Run("nothing_to_update_status", func(t *testing.T) {
		var statusMsg string
		onStatus := func(msg string) { statusMsg = msg }

		registry := &testutil.MockRegistry{
			GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, _ *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
				return domain.Extension{ID: id}, domain.DownloadInfo{
					ID:       id,
					Version:  goOldVer,
					Platform: domain.LinuxX64,
					Source:   "https://example.com/go.vsix",
				}, nil
			},
		}
		storage := &testutil.MockStorage{
			ListFunc: func(_ context.Context) ([]domain.Extension, error) {
				return []domain.Extension{goInstalled}, nil
			},
		}
		svc := NewUseCaseService(registry, storage, onStatus, 1)

		report, err := svc.Update(t.Context(), nil, UpdateOpts{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Results) != 0 {
			t.Fatalf("expected empty results, got %v", report.Results)
		}
		if statusMsg != "nothing to update" {
			t.Errorf("onStatus got %q, want %q", statusMsg, "nothing to update")
		}
	})

	t.Run("context_cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		registry := &testutil.MockRegistry{
			GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
				cancel()
				return domain.Extension{}, domain.DownloadInfo{}, context.Canceled
			},
		}
		storage := &testutil.MockStorage{
			ListFunc: func(_ context.Context) ([]domain.Extension, error) {
				return []domain.Extension{goInstalled}, nil
			},
		}
		svc := NewUseCaseService(registry, storage, nil, 1)

		_, err := svc.Update(ctx, []domain.ExtensionID{goID}, noopUpdateOpts())

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got error %v, want context.Canceled", err)
		}
	})
}
