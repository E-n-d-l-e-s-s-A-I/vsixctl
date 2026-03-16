package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type UpdateOpts struct {
	Confirm           func(toUpdate []domain.UpdateInfo) bool
	OnProgressFactory OnProgressFactory
}

type UpdateReport struct {
	Results []domain.ExtensionResult
}

// Update обновляет расширения
func (s *UseCaseService) Update(ctx context.Context, ids []domain.ExtensionID, opts UpdateOpts) (UpdateReport, error) {
	s.onStatus("search for updates...")
	resolved, skipped, err := s.updateResolve(ctx, ids)
	if err != nil {
		return UpdateReport{}, fmt.Errorf("update: %w", err)
	}

	if len(resolved) == 0 {
		s.onStatus("nothing to update")
		return UpdateReport{Results: skipped}, nil
	}

	if ok := opts.Confirm(resolved); !ok {
		return UpdateReport{Results: skipped}, nil
	}

	updateResults := s.update(ctx, resolved, opts.OnProgressFactory)
	results := append(skipped, updateResults...)

	return UpdateReport{Results: results}, nil
}

func (s *UseCaseService) updateResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.UpdateInfo, skipped []domain.ExtensionResult, err error) {
	installed, err := s.storage.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("update resolve: %w", err)
	}
	idToInstalled := make(map[domain.ExtensionID]domain.Extension, len(installed))
	for _, ext := range installed {
		idToInstalled[ext.ID] = ext
	}

	var requested []domain.Extension
	if len(ids) != 0 {
		for _, id := range ids {
			installedExt, ok := idToInstalled[id]
			if !ok {
				skipped = append(skipped, domain.ExtensionResult{ID: id, Err: domain.ErrNotInstalled})
				continue
			}
			requested = append(requested, installedExt)
		}
	} else {
		requested = installed
	}

	// Параллельный резолв каждого расширения поштучно
	type resolveResult struct {
		ext      domain.Extension
		download domain.DownloadInfo
		err      error
	}

	results := make([]resolveResult, len(requested))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.parallelism)

	for i, ext := range requested {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				_, download, err := s.registry.GetDownloadInfo(ctx, ext.ID, nil)
				results[i] = resolveResult{ext: ext, download: download, err: err}
			case <-ctx.Done():
				results[i] = resolveResult{ext: ext, err: ctx.Err()}
			}
		}()
	}
	wg.Wait()

	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	for _, r := range results {
		if r.err != nil {
			skipped = append(skipped, domain.ExtensionResult{ID: r.ext.ID, Err: r.err})
			continue
		}
		if !r.download.Version.NewerThan(r.ext.Version) {
			continue
		}
		resolved = append(resolved, domain.UpdateInfo{Prev: r.ext, New: r.download})
	}

	return resolved, skipped, nil
}

func (s *UseCaseService) update(ctx context.Context, resolved []domain.UpdateInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.parallelism)
	results := make([]domain.ExtensionResult, len(resolved))

	for i, ext := range resolved {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(ext.New.ID.String(), ext.New.Size)
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				vsix, err := s.registry.Download(ctx, ext.New, onProgress)
				if err != nil {
					results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: err}
					return
				}
				err = s.storage.Update(ctx, ext.New.ID, ext.New.Version, ext.New.Platform, vsix)
				results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: err}
			case <-ctx.Done():
				results[i] = domain.ExtensionResult{ID: ext.Prev.ID, Err: ctx.Err()}
			}
		}()
	}
	wg.Wait()

	return results
}
