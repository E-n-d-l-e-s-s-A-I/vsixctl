package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/testutil"
)

// noopOpts - InstallOpts, которые всегда подтверждают и не отслеживают прогресс
func noopOpts() InstallOpts {
	return InstallOpts{
		Confirm:           func([]domain.ExtensionID, []domain.DownloadInfo, []domain.ReinstallInfo) bool { return true },
		OnProgressFactory: func(string, int64) (domain.ProgressFunc, func()) { return func(int64) {}, func() {} },
	}
}

func rejectOpts() InstallOpts {
	opts := noopOpts()
	opts.Confirm = func([]domain.ExtensionID, []domain.DownloadInfo, []domain.ReinstallInfo) bool { return false }
	return opts
}

func TestInstall(t *testing.T) {
	goID := domain.ExtensionID{Publisher: "golang", Name: "go"}
	pythonID := domain.ExtensionID{Publisher: "ms-python", Name: "python"}
	depID := domain.ExtensionID{Publisher: "some", Name: "dep"}
	builtinID := domain.ExtensionID{Publisher: "vscode", Name: "builtin"}

	goDownload := domain.DownloadInfo{
		ID:       goID,
		Version:  domain.Version{Major: 0, Minor: 53, Patch: 1},
		Platform: domain.LinuxX64,
		Source:   "https://example.com/go.vsix",
	}
	pythonDownload := domain.DownloadInfo{
		ID:       pythonID,
		Version:  domain.Version{Major: 2026, Minor: 2, Patch: 0},
		Platform: domain.LinuxX64,
		Source:   "https://example.com/python.vsix",
	}
	depDownload := domain.DownloadInfo{
		ID:       depID,
		Version:  domain.Version{Major: 1, Minor: 0, Patch: 0},
		Platform: domain.LinuxX64,
		Source:   "https://example.com/dep.vsix",
	}

	goExt := domain.Extension{ID: goID}
	pythonExt := domain.Extension{ID: pythonID}
	// расширение с зависимостями: одна обычная + одна built-in (должна быть пропущена)
	goExtWithDeps := domain.Extension{
		ID:           goID,
		Dependencies: []domain.ExtensionID{depID, builtinID},
	}
	depExt := domain.Extension{ID: depID}

	connRefused := errors.New("connection refused")
	diskFull := errors.New("disk full")

	tests := []struct {
		name            string
		ids             []domain.InstallTarget
		opts            InstallOpts
		registry        *testutil.MockRegistry
		storage         *testutil.MockStorage
		want            []domain.ExtensionResult
		wantErr         error
		wantNewInstalls int // ожидаемое кол-во toInstall в Confirm (-1 = не проверять)
		wantReinstalls  int // ожидаемое кол-во toReinstall в Confirm (-1 = не проверять)
	}{
		{
			name: "single_extension",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}},
		},
		{
			name: "specific_version",
			ids: []domain.InstallTarget{
				{ID: goID, Version: &domain.Version{Major: 0, Minor: 50, Patch: 0}},
			},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					if version == nil {
						t.Error("expected non-nil version for requested extension")
					}
					if *version != (domain.Version{Major: 0, Minor: 50, Patch: 0}) {
						t.Errorf("version: got %v, want 0.50.0", version)
					}
					return goExt, domain.DownloadInfo{
						ID:      goID,
						Version: *version,
						Source:  "https://example.com/go.vsix",
					}, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}},
		},
		{
			name: "mixed_versioned_and_latest",
			ids: []domain.InstallTarget{
				{ID: goID, Version: &domain.Version{Major: 0, Minor: 50, Patch: 0}},
				{ID: pythonID},
			},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						if version == nil {
							t.Error("expected non-nil version for goID")
						}
						return goExt, domain.DownloadInfo{
							ID:      goID,
							Version: *version,
							Source:  "https://example.com/go.vsix",
						}, nil
					case pythonID:
						if version != nil {
							t.Error("expected nil version for pythonID")
						}
						return pythonExt, pythonDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID}},
		},
		{
			name: "already_installed",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{{ID: goID}}, nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: domain.ErrAlreadyInstalled}},
		},
		{
			name: "all_already_installed",
			ids:  []domain.InstallTarget{{ID: goID}, {ID: pythonID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExt, goDownload, nil
					case pythonID:
						return pythonExt, pythonDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{{ID: goID}, {ID: pythonID}}, nil
				},
			},
			want: []domain.ExtensionResult{
				{ID: goID, Err: domain.ErrAlreadyInstalled},
				{ID: pythonID, Err: domain.ErrAlreadyInstalled},
			},
		},
		{
			name: "with_dependencies",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExtWithDeps, goDownload, nil
					case depID:
						return depExt, depDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}},
		},
		{
			name: "confirm_rejected",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: rejectOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
			},
			want: nil,
		},
		{
			name: "resolve_error",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return domain.Extension{}, domain.DownloadInfo{}, connRefused
				},
			},
			wantErr: connRefused,
		},
		{
			name: "resolve_error_with_one_extension",
			ids:  []domain.InstallTarget{{ID: goID}, {ID: pythonID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExt, goDownload, nil
					case pythonID:
						return domain.Extension{}, domain.DownloadInfo{}, domain.ErrAllSourcesUnavailable
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
			},
			wantErr: domain.ErrAllSourcesUnavailable,
		},
		{
			name: "download_error",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return nil, domain.ErrAllSourcesUnavailable
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: domain.ErrAllSourcesUnavailable}},
		},
		{
			name: "download_error_with_one_ext",
			ids:  []domain.InstallTarget{{ID: goID}, {ID: pythonID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExt, goDownload, nil
					case pythonID:
						return pythonExt, pythonDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, info domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					if info.ID == pythonID {
						return nil, domain.ErrAllSourcesUnavailable
					}
					return []byte("vsix-data"), nil

				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID, Err: domain.ErrAllSourcesUnavailable}},
		},
		{
			name: "storage_install_error",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return diskFull
				},
			},
			want: []domain.ExtensionResult{{ID: goID, Err: diskFull}},
		},
		{
			name: "storage_install_error_with_one_ext",
			ids:  []domain.InstallTarget{{ID: goID}, {ID: pythonID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExt, goDownload, nil
					case pythonID:
						return pythonExt, pythonDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, id domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					if id == pythonID {
						return diskFull
					}
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID, Err: diskFull}},
		},
		{
			name: "extension_pack",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID, ExtensionPack: []domain.ExtensionID{depID}}, goDownload, nil
					case depID:
						return depExt, depDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}},
		},
		{
			name: "shared_dependency",
			ids:  []domain.InstallTarget{{ID: goID}, {ID: pythonID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID, Dependencies: []domain.ExtensionID{depID}}, goDownload, nil
					case pythonID:
						return domain.Extension{ID: pythonID, Dependencies: []domain.ExtensionID{depID}}, pythonDownload, nil
					case depID:
						return depExt, depDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: pythonID}, {ID: depID}},
		},
		{
			name: "circular_dependency",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return domain.Extension{ID: goID, Dependencies: []domain.ExtensionID{depID}}, goDownload, nil
					case depID:
						return domain.Extension{ID: depID, ExtensionPack: []domain.ExtensionID{goID}}, depDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want: []domain.ExtensionResult{{ID: goID}, {ID: depID}},
		},
		{
			name: "force_reinstalls",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: func() InstallOpts {
				opts := noopOpts()
				opts.Force = true
				return opts
			}(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{{ID: goID}}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want:            []domain.ExtensionResult{{ID: goID}},
			wantNewInstalls: 0,
			wantReinstalls:  1,
		},
		{
			name: "force_skips_installed_dependencies",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: func() InstallOpts {
				opts := noopOpts()
				opts.Force = true
				return opts
			}(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					switch id {
					case goID:
						return goExtWithDeps, goDownload, nil
					case depID:
						return depExt, depDownload, nil
					}
					return domain.Extension{}, domain.DownloadInfo{}, domain.ErrNotFound
				},
				DownloadFunc: func(_ context.Context, _ domain.DownloadInfo, _ domain.ProgressFunc) ([]byte, error) {
					return []byte("vsix-data"), nil
				},
			},
			storage: &testutil.MockStorage{
				ListFunc: func(_ context.Context) ([]domain.Extension, error) {
					return []domain.Extension{{ID: goID}, {ID: depID}}, nil
				},
				InstallFunc: func(_ context.Context, _ domain.ExtensionID, _ domain.Version, _ domain.Platform, _ []byte) error {
					return nil
				},
			},
			want:            []domain.ExtensionResult{{ID: goID}},
			wantNewInstalls: 0,
			wantReinstalls:  1,
		},
		{
			name: "storage_list_error",
			ids:  []domain.InstallTarget{{ID: goID}},
			opts: noopOpts(),
			registry: &testutil.MockRegistry{
				GetDownloadInfoFunc: func(_ context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
					return goExt, goDownload, nil
				},
			},
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
			opts := testCase.opts

			// Оборачиваем Confirm для проверки аргументов
			var gotNewInstalls, gotReinstalls int
			if testCase.wantNewInstalls != 0 || testCase.wantReinstalls != 0 {
				origConfirm := opts.Confirm
				opts.Confirm = func(ids []domain.ExtensionID, toInstall []domain.DownloadInfo, toReinstall []domain.ReinstallInfo) bool {
					gotNewInstalls = len(toInstall)
					gotReinstalls = len(toReinstall)
					return origConfirm(ids, toInstall, toReinstall)
				}
			}

			svc := NewUseCaseService(testCase.registry, testCase.storage, nil, 1)

			report, err := svc.Install(t.Context(), testCase.ids, opts)

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
			if testCase.wantNewInstalls != 0 || testCase.wantReinstalls != 0 {
				if gotNewInstalls != testCase.wantNewInstalls {
					t.Errorf("confirm toInstall: got %d, want %d", gotNewInstalls, testCase.wantNewInstalls)
				}
				if gotReinstalls != testCase.wantReinstalls {
					t.Errorf("confirm toReinstall: got %d, want %d", gotReinstalls, testCase.wantReinstalls)
				}
			}
			assertResults(t, report.Results, testCase.want)
		})
	}

	t.Run("context_cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		registry := &testutil.MockRegistry{
			GetDownloadInfoFunc: func(_ context.Context, _ domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
				cancel()
				return domain.Extension{}, domain.DownloadInfo{}, context.Canceled
			},
		}
		svc := NewUseCaseService(registry, nil, nil, 1)

		_, err := svc.Install(ctx, []domain.InstallTarget{{ID: goID}}, noopOpts())

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got error %v, want context.Canceled", err)
		}
	})
}

// assertResults сравнивает результаты без учёта порядка (из-за конкурентного выполнения)
func assertResults(t *testing.T, got, want []domain.ExtensionResult) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	matched := make([]bool, len(want))
	for _, g := range got {
		found := false
		for i, w := range want {
			if matched[i] {
				continue
			}
			if g.ID == w.ID && errorsEqual(g.Err, w.Err) {
				matched[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected result: %+v\nwant: %+v", g, want)
		}
	}
}

// errorsEqual сравнивает ошибки через errors.Is, с fallback на сравнение текста
func errorsEqual(a, b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if errors.Is(a, b) {
		return true
	}
	return a.Error() == b.Error()
}
