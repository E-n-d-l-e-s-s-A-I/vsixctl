package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/httputil"
)

const (
	VsixAssetPath         = "/Microsoft.VisualStudio.Services.VSIXPackage"
	DependenciesProperty  = "Microsoft.VisualStudio.Code.ExtensionDependencies"
	ExtensionPackProperty = "Microsoft.VisualStudio.Code.ExtensionPack"
	EngineProperty        = "Microsoft.VisualStudio.Code.Engine"
)

type Registry struct {
	url               string
	client            *http.Client
	platform          domain.Platform // Платформа на которой запущена утилита
	vscodeVer         domain.Version  // Версия vscode на устройстве
	sourceIdleTimeout time.Duration   // Таймаут на ответ источника при скачивании. По истечении таймаута переходим к следующему источнику
	queryTimeout      time.Duration   // Таймаут на запросы к API маркетплейса (поиск, получение метаданных)
	queryRetries      int             // Кол-во ретраев на запросы к marketplace
	logger            domain.Logger   // Логгер
}

const (
	DefaultURL                    = "https://marketplace.visualstudio.com/_apis/public/gallery"
	DefaultMaxIdleConns           = 100
	DefaultMaxConnsPerHost        = 10
	DefaultIdleConnTimeout        = 90 * time.Second
	DefaultSHandshakeTimeout      = 3 * time.Second
	DefaultSResponseHeaderTimeout = 4 * time.Second
	DefaultTimeout                = 3 * time.Minute
)

func NewRegistry(url string, client *http.Client, vscodeVer domain.Version, platform domain.Platform, sourceIdleTimeout time.Duration, queryTimeout time.Duration, queryRetries int, l domain.Logger) *Registry {
	if l == nil {
		l = domain.NopLogger()
	}
	return &Registry{
		url:               url,
		client:            client,
		platform:          platform,
		vscodeVer:         vscodeVer,
		sourceIdleTimeout: sourceIdleTimeout,
		queryTimeout:      queryTimeout,
		queryRetries:      queryRetries,
		logger:            l,
	}
}

func NewDefaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          DefaultMaxIdleConns,
			MaxConnsPerHost:       DefaultMaxConnsPerHost,
			IdleConnTimeout:       DefaultIdleConnTimeout,
			TLSHandshakeTimeout:   DefaultSHandshakeTimeout,
			ResponseHeaderTimeout: DefaultSResponseHeaderTimeout,
			Proxy:                 http.ProxyFromEnvironment,
		},
		Timeout: DefaultTimeout,
	}
}

func (r *Registry) Search(ctx context.Context, query domain.SearchQuery) ([]domain.Extension, error) {
	filterType, ok := searchTypeToFilterType[query.Type]
	if !ok {
		return nil, fmt.Errorf("search: unexpected search type")
	}

	searchRequest := searchRequest{
		Filters: []searchFilter{
			{
				Criteria: []searchCriteria{
					{
						FilterType: filterType,
						Value:      query.Query,
					},
				},
				PageNumber: 1,
				PageSize:   query.Limit,
				SortBy:     SortByRelevance,
				SortOrder:  SortOrderDefault,
			},
		},
		AssetTypes: []string{},
		Flags:      FlagIncludeStatistics,
	}
	searchResponse, err := r.extensionQuery(ctx, searchRequest, fmt.Sprintf("search %q", query.Query))
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}
	if len(searchResponse.Results) == 0 {
		return []domain.Extension{}, nil
	}

	responseResult := searchResponse.Results[0]
	var result []domain.Extension
	for _, extension := range responseResult.Extensions {
		domainExt, err := r.marketplaceExtensionToDomain(extension, Version{Version: "0.0.0"}) // заглушка, т.к. версия при поиске не важна
		if err != nil {
			continue
		}
		result = append(result, domainExt)
	}

	return result, nil
}

func (r *Registry) getExtension(ctx context.Context, id domain.ExtensionID, flags int, label string) (Extension, error) {
	searchRequest := searchRequest{
		Filters: []searchFilter{
			{
				Criteria: []searchCriteria{
					{
						FilterType: ExtensionIdSearch,
						Value:      id.String(),
					},
				},
				PageNumber: 1,
				PageSize:   1,
				SortBy:     SortByRelevance,
				SortOrder:  SortOrderDefault,
			},
		},
		AssetTypes: []string{},
		Flags:      flags,
	}
	searchResponse, err := r.extensionQuery(ctx, searchRequest, fmt.Sprintf("%s %s", id.String(), label))
	if err != nil {
		return Extension{}, fmt.Errorf("get extension: %w", err)
	}
	if len(searchResponse.Results) < 1 {
		return Extension{}, fmt.Errorf("get extension: %w", domain.ErrNotFound)
	}
	if len(searchResponse.Results[0].Extensions) < 1 {
		return Extension{}, fmt.Errorf("get extension: %w", domain.ErrNotFound)
	}
	return searchResponse.Results[0].Extensions[0], nil
}

func (r *Registry) GetDownloadInfo(ctx context.Context, id domain.ExtensionID, version *domain.Version) (domain.Extension, domain.DownloadInfo, error) {
	flags := baseFlags | FlagIncludeVersions

	var (
		releaseVersion Version
		found          bool
		extension      Extension
		err            error
	)

	if version == nil {
		// Ищем последнюю версию => сначала делаем запрос на последнюю версию
		extension, err = r.getExtension(ctx, id, flags|FlagIncludeLatestOnly, "get latest version")
		if err != nil {
			return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
		}

		releaseVersion, found = findLatestSupportedVersion(extension.Versions, r.vscodeVer, r.platform)
		if !found {
			// Если последняя версия не совместима => делаем запрос на все версии
			extension, err = r.getExtension(ctx, id, flags, "get all versions")
			if err != nil {
				return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
			}
			releaseVersion, found = findLatestSupportedVersion(extension.Versions, r.vscodeVer, r.platform)
		}
	} else {
		// Ищем специфичную версию => делаем запрос на все версии
		extension, err = r.getExtension(ctx, id, flags, "get all versions")
		if err != nil {
			return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
		}
		releaseVersion, found = findSpecificSupportedVersion(extension.Versions, *version, r.vscodeVer, r.platform)
	}

	if !found {
		return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", domain.ErrVersionNotFound)
	}
	domainExt, err := r.marketplaceExtensionToDomain(extension, releaseVersion)
	if err != nil {
		return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
	}

	// Прямая ссылка на скачивание
	directUri := fmt.Sprintf("%s/publishers/%s/vsextensions/%s/%s/vspackage", r.url, id.Publisher, id.Name, domainExt.Version.String())
	if releaseVersion.TargetPlatform != "" {
		directUri += fmt.Sprintf("?targetPlatform=%s", releaseVersion.TargetPlatform)
	}

	mainSource := releaseVersion.AssetUri + VsixAssetPath
	fallBackSource := releaseVersion.FallbackAssetUri + VsixAssetPath

	size := r.getSize(ctx, []string{mainSource, fallBackSource, directUri})

	hasPreRelease := false
	for _, v := range extension.Versions {
		if isPreRelease(v) {
			hasPreRelease = true
			break
		}
	}

	return domainExt,
		domain.DownloadInfo{
			ID:       id,
			Version:  domainExt.Version,
			Platform: domain.Platform(releaseVersion.TargetPlatform),
			Size:     size,
			Source:   mainSource,
			Meta: domain.ExtensionMeta{
				UUID:                 extension.ExtensionId,
				PublisherID:          extension.Publisher.PublisherId,
				PublisherDisplayName: extension.Publisher.DisplayName,
				IsPreRelease:         isPreRelease(releaseVersion),
				HasPreRelease:        hasPreRelease,
			},
			FallbackSources: []string{fallBackSource, directUri},
		}, nil
}

// Скачивание vsix-пакета, учитывает разные источники
// Если источник недоступен переходит к следующему
func (r *Registry) Download(ctx context.Context, info domain.DownloadInfo, onProgress domain.ProgressFunc) ([]byte, error) {
	// Формирование списка источников
	sources := append([]string{info.Source}, info.FallbackSources...)

	// Пытаемся скачать расширение с одного из источников
	// Если источник долго не отвечает, переходим на следующий
	for i, source := range sources {
		data, err := r.downloadFromSource(ctx, source, onProgress)
		if err != nil {
			// Если ошибка не от downloadFromSource - выходим
			if ctx.Err() != nil {
				return nil, fmt.Errorf("download: %w", ctx.Err())
			}
			logErr := err
			if errors.Is(err, httputil.ErrStalled) {
				logErr = fmt.Errorf("source timed out")
			}
			r.logger.Warn("[%s download] source %d/%d unavailable: %v", info.ID, i+1, len(sources), logErr)
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("download: %w", domain.ErrAllSourcesUnavailable)
}

// Получает версии расширения
func (r *Registry) GetVersions(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
	extension, err := r.getExtension(ctx, id, baseFlags|FlagIncludeVersions, "get all versions")
	if err != nil {
		return nil, fmt.Errorf("get versions: %w", err)
	}

	// Группировка вариантов по номеру версии с сохранением порядка API (newest first)
	type versionGroup struct {
		version  domain.Version
		variants []Version
	}
	seen := make(map[domain.Version]int) // версия → индекс в groups
	var groups []versionGroup
	for _, ver := range extension.Versions {
		domainVer, err := domain.ParseVersion(ver.Version)
		if err != nil {
			r.logger.Warn("parse version %q: %v", ver.Version, err)
			continue
		}
		if idx, ok := seen[domainVer]; ok {
			groups[idx].variants = append(groups[idx].variants, ver)
		} else {
			seen[domainVer] = len(groups)
			groups = append(groups, versionGroup{version: domainVer, variants: []Version{ver}})
		}
	}

	var result []domain.VersionInfo
	for _, group := range groups {
		vscodeCompatible := false
		platformCompatible := false
		hasStableVariant := false

		for _, variant := range group.variants {
			if isPreRelease(variant) {
				continue
			}
			hasStableVariant = true
			if isEngineCompatible(r.vscodeVer, findProperty(variant.Properties, EngineProperty)) {
				vscodeCompatible = true
			}
			if variant.TargetPlatform == string(r.platform) || variant.TargetPlatform == "" {
				platformCompatible = true
			}
		}
		// Пропускаем версии у которых все варианты - pre-release
		if !hasStableVariant {
			continue
		}

		result = append(result, domain.VersionInfo{
			Version:            group.version,
			VscodeCompatible:   vscodeCompatible,
			PlatformCompatible: platformCompatible,
		})
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

// findLatestSupportedVersion находит последнюю релизную версию расширения,
// совместимую с версией vscode и платформой.
// Platform-specific версия имеет приоритет над универсальной.
func findLatestSupportedVersion(versions []Version, vscodeVer domain.Version, platform domain.Platform) (Version, bool) {
	var universalFallback *Version
	for _, v := range versions {
		if isPreRelease(v) {
			continue
		}
		if !isEngineCompatible(vscodeVer, findProperty(v.Properties, EngineProperty)) {
			continue
		}
		if v.TargetPlatform == string(platform) {
			return v, true
		}
		if v.TargetPlatform == "" && universalFallback == nil {
			universalFallback = &v
		}
	}
	if universalFallback != nil {
		return *universalFallback, true
	}
	return Version{}, false
}

// findSpecificSupportedVersion находит определённую версию расширения,
// совместимую с версией vscode и платформой.
// Platform-specific версия имеет приоритет над универсальной.
func findSpecificSupportedVersion(versions []Version, version domain.Version, vscodeVer domain.Version, platform domain.Platform) (Version, bool) {
	var universalFallback *Version
	for _, v := range versions {
		if isPreRelease(v) {
			continue
		}
		if !isEngineCompatible(vscodeVer, findProperty(v.Properties, EngineProperty)) {
			continue
		}
		parsedVer, err := domain.ParseVersion(v.Version)
		if err != nil || parsedVer != version {
			continue
		}

		if v.TargetPlatform == string(platform) {
			return v, true
		}
		if v.TargetPlatform == "" && universalFallback == nil {
			universalFallback = &v
		}
	}
	if universalFallback != nil {
		return *universalFallback, true
	}
	return Version{}, false
}

// isEngineCompatible проверяет, совместима ли версия vscode с engine constraint расширения.
// Поддерживаемые форматы: "^1.107.0", ">=1.80.0", "~1.80.0", "1.80.0", "*", "".
func isEngineCompatible(vscodeVer domain.Version, engine string) bool {
	engine = strings.TrimSpace(engine)
	if engine == "" || engine == "*" {
		return true
	}
	engine = strings.TrimLeft(engine, "^>=~")
	// Отсекаем pre-release суффикс (например, "1.110.0-20260204" → "1.110.0")
	if idx := strings.IndexByte(engine, '-'); idx != -1 {
		engine = engine[:idx]
	}
	minVer, err := domain.ParseVersion(engine)
	if err != nil {
		return false
	}
	return vscodeVer == minVer || vscodeVer.NewerThan(minVer)
}

func isPreRelease(v Version) bool {
	for _, p := range v.Properties {
		if p.Key == "Microsoft.VisualStudio.Code.PreRelease" && p.Value == "true" {
			return true
		}
	}
	return false
}

// queryError оборачивает ошибку запроса с признаком возможности повторной попытки
type queryError struct {
	err       error
	retryable bool
}

func (e *queryError) Error() string { return e.err.Error() }
func (e *queryError) Unwrap() error { return e.err }

// extensionQuery выполняет запрос к API маркетплейса с ретраями
func (r *Registry) extensionQuery(ctx context.Context, searchReq searchRequest, label string) (SearchResponse, error) {
	totalAttempts := r.queryRetries + 1
	var lastErr error
	for attempt := range totalAttempts {
		if ctx.Err() != nil {
			return SearchResponse{}, fmt.Errorf("extension query: %w", ctx.Err())
		}

		queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
		resp, err := r.doExtensionQuery(queryCtx, searchReq)
		cancel()

		if err == nil {
			return resp, nil
		}

		var qe *queryError
		if errors.As(err, &qe) && !qe.retryable {
			return SearchResponse{}, qe.err
		}

		lastErr = err
		logErr := err
		if errors.Is(err, context.DeadlineExceeded) {
			var urlErr *url.Error
			if errors.As(err, &urlErr) {
				logErr = fmt.Errorf("%s %q: query timed out", urlErr.Op, urlErr.URL)
			} else {
				logErr = fmt.Errorf("query timed out")
			}
		}
		r.logger.Warn("[%s] attempt %d/%d failed: %v", label, attempt+1, totalAttempts, logErr)
	}
	return SearchResponse{}, lastErr
}

func (r *Registry) doExtensionQuery(ctx context.Context, searchRequest searchRequest) (SearchResponse, error) {
	body, err := json.Marshal(searchRequest)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("extension query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/extensionquery", r.url), bytes.NewBuffer(body))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("extension query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;api-version=7.1-preview.1")

	resp, err := r.client.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("extension query: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SearchResponse{}, &queryError{
			err:       fmt.Errorf("extension query: unexpected response status code %d", resp.StatusCode),
			retryable: resp.StatusCode >= 500,
		}
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("extension query: %w", err)
	}
	searchResponse := SearchResponse{}
	err = json.Unmarshal(bytes, &searchResponse)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("extension query: %w", err)
	}
	return searchResponse, nil
}

// parseExtensionIDs парсит строку вида "publisher1.ext1,publisher2.ext2" в слайс ExtensionID
func parseExtensionIDs(raw string) ([]domain.ExtensionID, error) {
	if raw == "" {
		return nil, nil
	}
	var ids []domain.ExtensionID
	for _, rawID := range strings.Split(raw, ",") {
		if rawID == "" {
			continue
		}
		id, err := domain.ParseExtensionID(rawID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// findProperty ищет значение свойства по ключу в списке Properties
func findProperty(properties []Property, key string) string {
	for _, p := range properties {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// downloadFromSource скачивает расширение из источника(ссылки) source
func (r *Registry) downloadFromSource(ctx context.Context, source string, onProgress domain.ProgressFunc) ([]byte, error) {
	// Контекст для отмены запроса при таймауте ожидания headers
	// Нельзя использовать context.WithTimeout — cancel() убьёт и чтение body
	// ResponseHeaderTimeout не работает с HTTP/2, поэтому гоняем Do против таймера
	reqCtx, reqCancel := context.WithCancel(ctx)
	defer reqCancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("download from source: %w", err)
	}

	type doResult struct {
		resp *http.Response
		err  error
	}
	ch := make(chan doResult, 1)
	go func() {
		resp, err := r.client.Do(req)
		ch <- doResult{resp, err}
	}()

	var resp *http.Response
	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("download from source: %w", res.err)
		}
		resp = res.resp
	case <-time.After(r.queryTimeout):
		reqCancel()
		<-ch // ждём завершения горутины
		return nil, fmt.Errorf("download from source: awaiting headers timed out")
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download from source: unexpected response status code %d", resp.StatusCode)
	}

	reader := httputil.NewProgressReader(httputil.NewStallReader(resp.Body, r.sourceIdleTimeout), onProgress)
	return io.ReadAll(reader)
}

// getSize получает размер расширения, через Head или GET запрос.
func (r *Registry) getSize(ctx context.Context, sources []string) int64 {
	for _, source := range sources {
		size, err := r.getSizeHeadRequest(ctx, source)
		if err == nil {
			return size
		}
		size, err = r.getSizeGetRequest(ctx, source)
		if err == nil {
			return size
		}
	}
	r.logger.Warn("get size: all sources unavailable")
	return 0
}

// getSizeHeadRequest делает Head-запрос для получения размера расширения
func (r *Registry) getSizeHeadRequest(ctx context.Context, source string) (int64, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, source, nil)
	if err != nil {
		return 0, fmt.Errorf("get size head request: %v", err)
	}
	resp, err := r.client.Do(req)

	if err != nil {
		return 0, fmt.Errorf("get size head request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get size head request: unexpected status: %d", resp.StatusCode)
	}
	if resp.ContentLength == -1 {
		return 0, fmt.Errorf("get size head request: unknown size")
	}
	return resp.ContentLength, nil
}

// getSizeGetRequest делает Get-запрос для получения размера расширения
func (r *Registry) getSizeGetRequest(ctx context.Context, source string) (int64, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, source, nil)
	if err != nil {
		return 0, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("get size get request: unexpected status: %d", resp.StatusCode)
	}
	if resp.ContentLength == -1 {
		return 0, fmt.Errorf("get size get request: unknown size")
	}
	return resp.ContentLength, nil
}

// marketplaceExtensionToDomain приводит модель расширения маркетплейса в доменную модель
func (r *Registry) marketplaceExtensionToDomain(ext Extension, releaseVersion Version) (domain.Extension, error) {
	version, err := domain.ParseVersion(releaseVersion.Version)
	if err != nil {
		return domain.Extension{}, fmt.Errorf("marketplace extension to domain: %w", err)
	}
	extensionPack, err := parseExtensionIDs(findProperty(releaseVersion.Properties, ExtensionPackProperty))
	if err != nil {
		return domain.Extension{}, fmt.Errorf("marketplace extension to domain: %w", err)
	}
	dependencies, err := parseExtensionIDs(findProperty(releaseVersion.Properties, DependenciesProperty))
	if err != nil {
		return domain.Extension{}, fmt.Errorf("marketplace extension to domain: %w", err)
	}

	return domain.Extension{
		ID: domain.ExtensionID{
			Name:      ext.ExtensionName,
			Publisher: ext.Publisher.PublisherName,
		},
		Description:   ext.ShortDescription,
		Version:       version,
		Dependencies:  dependencies,
		ExtensionPack: extensionPack,
	}, nil

}
