package usecases

import (
	"context"
	"fmt"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type RemoveOpts struct {
	Confirm func(requestedIDs []domain.ExtensionID, extensions []domain.Extension) bool
}

type RemoveReport struct {
	Results []domain.ExtensionResult
}

// Remove удаляет расширения
func (s *UseCaseService) Remove(ctx context.Context, ids []domain.ExtensionID, opts RemoveOpts) (RemoveReport, error) {
	resolved, notInstalled, err := s.removeResolve(ctx, ids)
	if err != nil {
		return RemoveReport{}, fmt.Errorf("remove: %w", err)
	}

	// Формируем результаты для неустановленных расширений
	var results []domain.ExtensionResult
	for _, id := range notInstalled {
		results = append(results, domain.ExtensionResult{ID: id, Err: domain.ErrNotInstalled})
	}
	if len(resolved) == 0 {
		return RemoveReport{Results: results}, nil
	}

	if ok := opts.Confirm(ids, resolved); !ok {
		return RemoveReport{Results: results}, nil
	}

	for _, ext := range resolved {
		err := s.storage.Remove(ctx, ext.ID)
		results = append(results, domain.ExtensionResult{ID: ext.ID, Err: err})
	}

	return RemoveReport{Results: results}, nil
}

// removeResolve резолвит все удаляемые расширения
// Добавляет к самим расширениям их пакетные расширения, а так же фильтрует не установленные
func (s *UseCaseService) removeResolve(ctx context.Context, ids []domain.ExtensionID) (resolved []domain.Extension, notInstalled []domain.ExtensionID, err error) {
	installed, err := s.List(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve remove: %w", err)
	}
	installedMap := make(map[domain.ExtensionID]domain.Extension, len(installed))
	for _, ext := range installed {
		installedMap[ext.ID] = ext
	}

	for _, id := range ids {
		pack := resolvePack(id, installedMap)
		if len(pack) == 0 {
			notInstalled = append(notInstalled, id)
			continue
		}
		resolved = append(resolved, pack...)
	}
	resolved = uniqExtensions(resolved)

	return resolved, notInstalled, nil
}

// resolvePack рекурсивно возвращает все расширения из пакета, которые установлены
func resolvePack(id domain.ExtensionID, installed map[domain.ExtensionID]domain.Extension) []domain.Extension {
	var result []domain.Extension
	seen := make(map[domain.ExtensionID]struct{})

	var resolve func(id domain.ExtensionID)
	resolve = func(id domain.ExtensionID) {
		if _, ok := seen[id]; ok {
			return
		}

		seen[id] = struct{}{}
		ext, ok := installed[id]
		if !ok {
			return
		}
		result = append(result, ext)
		for _, packExt := range ext.ExtensionPack {
			resolve(packExt)
		}
	}
	resolve(id)

	return result
}

// uniqExtensions дедуплицирует список расширений
func uniqExtensions(extensions []domain.Extension) []domain.Extension {
	var result []domain.Extension
	seen := make(map[domain.ExtensionID]struct{})

	for _, ext := range extensions {
		if _, ok := seen[ext.ID]; ok {
			continue
		}
		result = append(result, ext)
		seen[ext.ID] = struct{}{}
	}

	return result
}
