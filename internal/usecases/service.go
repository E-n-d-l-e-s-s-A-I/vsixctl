package usecases

import (
	"context"
	"slices"
	"sync"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type OnProgressFactory func(string) (domain.ProgressFunc, func())

type UseCase interface {
	Search(ctx context.Context, query string, count int) ([]domain.Extension, error)
	Install(ctx context.Context, extensions map[domain.ExtensionID]domain.VersionInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult
	Resolve(ctx context.Context, ids []domain.ExtensionID) (map[domain.ExtensionID]domain.VersionInfo, []domain.ExtensionResult, error)
	Remove(ctx context.Context, ids []domain.ExtensionID) []domain.ExtensionResult
	Update(ctx context.Context) error
	List(ctx context.Context) ([]domain.Extension, error)
}

type UseCaseService struct {
	registry    domain.Registry
	storage     domain.Storage
	parallelism int // Кол-во параллельных загрузок
}

func NewUserCaseService(registry domain.Registry, storage domain.Storage, parallelism int) *UseCaseService {
	return &UseCaseService{
		registry:    registry,
		storage:     storage,
		parallelism: parallelism,
	}
}

// Search поиск расширений
func (s *UseCaseService) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	return s.registry.Search(ctx, query, count)
}

// Remove удаление расширений
func (s *UseCaseService) Remove(ctx context.Context, ids []domain.ExtensionID) []domain.ExtensionResult {
	results := make([]domain.ExtensionResult, len(ids))
	for i, id := range ids {
		err := s.storage.Remove(ctx, id)
		results[i] = domain.ExtensionResult{ID: id, Err: err}
	}
	return results
}

// Resolve резолв всех расширений и их зависимостей
func (s *UseCaseService) Resolve(ctx context.Context, ids []domain.ExtensionID) (map[domain.ExtensionID]domain.VersionInfo, []domain.ExtensionResult, error) {
	resolved, errs := s.resolveAll(ctx, ids)
	if len(errs) != 0 {
		return nil, errs, nil
	}

	// Фильтрация уже установленных
	installed, err := s.filterInstalled(ctx, resolved, ids)
	if err != nil {
		return nil, nil, err
	}

	return resolved, installed, nil
}

func (s *UseCaseService) Install(ctx context.Context, extensions map[domain.ExtensionID]domain.VersionInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	return s.downloadAndInstall(ctx, extensions, onProgressFactory)
}

// filterInstalled удаляет из resolved уже установленные расширения.
// Для запрошенных пользователем возвращает InstallResult с ErrAlreadyInstalled.
func (s *UseCaseService) filterInstalled(ctx context.Context, resolved map[domain.ExtensionID]domain.VersionInfo, requestedIDs []domain.ExtensionID) ([]domain.ExtensionResult, error) {
	installedExtensions, err := s.storage.List(ctx)
	if err != nil {
		return nil, err
	}
	installedMap := make(map[domain.ExtensionID]struct{}, len(installedExtensions))
	for _, ext := range installedExtensions {
		installedMap[ext.ID] = struct{}{}
	}

	var results []domain.ExtensionResult
	for id := range resolved {
		if _, ok := installedMap[id]; ok {
			delete(resolved, id)
			if slices.Contains(requestedIDs, id) {
				results = append(results, domain.ExtensionResult{ID: id, Err: domain.ErrAlreadyInstalled})
			}
		}
	}
	return results, nil
}

// downloadAndInstall асинхронно скачивает и устанавливает расширения
func (s *UseCaseService) downloadAndInstall(ctx context.Context, extensions map[domain.ExtensionID]domain.VersionInfo, onProgressFactory OnProgressFactory) []domain.ExtensionResult {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		sem     = make(chan struct{}, s.parallelism)
		results []domain.ExtensionResult
	)
	for id, ver := range extensions {
		wg.Add(1)
		onProgress, exitFunc := onProgressFactory(id.String())
		go func() {
			defer wg.Done()
			defer exitFunc()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				res := s.installExtension(ctx, id, ver, onProgress)
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			case <-ctx.Done():
				mu.Lock()
				results = append(results, domain.ExtensionResult{ID: id, Err: ctx.Err()})
				mu.Unlock()
				return
			}
		}()
	}
	wg.Wait()
	return results
}

func (s *UseCaseService) Update(ctx context.Context) error {
	return nil
}

func (s *UseCaseService) List(ctx context.Context) ([]domain.Extension, error) {
	return s.storage.List(ctx)
}

func (s *UseCaseService) installExtension(ctx context.Context, id domain.ExtensionID, ver domain.VersionInfo, onProgress domain.ProgressFunc) domain.ExtensionResult {
	data, err := s.registry.Download(ctx, ver, onProgress)
	if err != nil {
		return domain.ExtensionResult{ID: id, Err: err}
	}

	err = s.storage.Install(ctx, id, ver, data)

	return domain.ExtensionResult{ID: id, Err: err}
}

// Резолвит зависимости всех переданных расширений
func (s *UseCaseService) resolveAll(ctx context.Context, ids []domain.ExtensionID) (map[domain.ExtensionID]domain.VersionInfo, []domain.ExtensionResult) {
	var (
		visited  sync.Map
		mu       sync.Mutex
		resolved = make(map[domain.ExtensionID]domain.VersionInfo)
		errs     []domain.ExtensionResult
		wg       sync.WaitGroup
		sem      = make(chan struct{}, s.parallelism)
	)

	var resolve func(domain.ExtensionID)
	resolve = func(id domain.ExtensionID) {
		defer wg.Done()
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			latestVer, err := s.registry.GetLatestVersion(ctx, id)
			if err != nil {
				mu.Lock()
				errs = append(errs, domain.ExtensionResult{ID: id, Err: err})
				mu.Unlock()
				return
			}

			mu.Lock()
			resolved[id] = latestVer
			mu.Unlock()

			for _, dep := range latestVer.ExtensionPack {
				if dep.Publisher == domain.BuiltInPublisher {
					continue
				}
				if _, loaded := visited.LoadOrStore(dep, struct{}{}); !loaded {
					wg.Add(1)
					go resolve(dep)
				}
			}
			for _, dep := range latestVer.Dependencies {
				if dep.Publisher == domain.BuiltInPublisher {
					continue
				}
				if _, loaded := visited.LoadOrStore(dep, struct{}{}); !loaded && dep.Publisher != domain.BuiltInPublisher {
					wg.Add(1)
					go resolve(dep)
				}
			}

		case <-ctx.Done():
			mu.Lock()
			errs = append(errs, domain.ExtensionResult{ID: id, Err: ctx.Err()})
			mu.Unlock()
			return
		}
	}

	for _, id := range ids {
		if _, loaded := visited.LoadOrStore(id, struct{}{}); !loaded {
			wg.Add(1)
			go resolve(id)
		}
	}

	wg.Wait()
	return resolved, errs
}
