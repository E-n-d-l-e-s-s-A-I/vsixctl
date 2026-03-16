package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	url           string
	client        *http.Client
	platform      domain.Platform // Платформа на которой запущена утилита
	vscodeVer     domain.Version  // Версия vscode на устройстве
	sourceTimeout time.Duration   // Таймаут на ответ источника при скачивании. По истечении таймаута переходим к следующему источнику
	queryTimeout  time.Duration   // Таймаут на запросы к API маркетплейса (поиск, получение метаданных)
	logFunc       domain.LogFunc
}

const (
	DefaultURL                    = "https://marketplace.visualstudio.com/_apis/public/gallery"
	DefaultMaxIdleConns           = 100
	DefaultMaxConnsPerHost        = 10
	DefaultIdleConnTimeout        = 90 * time.Second
	DefaultSHandshakeTimeout      = 3 * time.Second
	DefaultSResponseHeaderTimeout = 4 * time.Second
	DefaultTimeout                = 3 * time.Minute
	DefaultQueryRetries           = 3
)

func NewRegistry(url string, client *http.Client, vscodeVer domain.Version, platform domain.Platform, sourceTimeout time.Duration, queryTimeout time.Duration, logFunc domain.LogFunc) *Registry {
	if logFunc == nil {
		logFunc = func(string) {}
	}
	return &Registry{
		url:           url,
		client:        client,
		platform:      platform,
		vscodeVer:     vscodeVer,
		sourceTimeout: sourceTimeout,
		queryTimeout:  queryTimeout,
		logFunc:       logFunc,
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
	searchResponse, err := r.extensionQuery(ctx, searchRequest)
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

func (r *Registry) getExtension(ctx context.Context, id domain.ExtensionID, flags int) (Extension, error) {
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
	searchResponse, err := r.extensionQuery(ctx, searchRequest)
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
		extension, err = r.getExtension(ctx, id, flags|FlagIncludeLatestOnly)
		if err != nil {
			return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
		}

		releaseVersion, found = findLatestSupportedVersion(extension.Versions, r.vscodeVer, r.platform)
		if !found {
			// Если последняя версия не совместима => делаем запрос на все версии
			extension, err = r.getExtension(ctx, id, flags)
			if err != nil {
				return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
			}
			releaseVersion, found = findLatestSupportedVersion(extension.Versions, r.vscodeVer, r.platform)
		}
	} else {
		// Ищем специфичную версию => делаем запрос на все версии
		extension, err = r.getExtension(ctx, id, flags)
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

	size, err := r.getSize(ctx, []string{mainSource, fallBackSource, directUri})
	if err != nil {
		return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get latest version: %w", err)
	}

	return domainExt,
		domain.DownloadInfo{
			ID:              id,
			Version:         domainExt.Version,
			Platform:        domain.Platform(releaseVersion.TargetPlatform),
			Size:            size,
			Source:          mainSource,
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
	for _, source := range sources {
		data, err := r.downloadFromSource(ctx, source, onProgress)
		if err != nil {
			// Если ошибка не от downloadFromSource - выходим
			if ctx.Err() != nil {
				return nil, fmt.Errorf("download: %w", ctx.Err())
			}
			r.logFunc(fmt.Sprintf("source %s unavailable: %v", source, err))
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("download: %w", domain.ErrAllSourcesUnavailable)
}

// Получает версии расширения
func (r *Registry) GetVersions(ctx context.Context, id domain.ExtensionID, limit int) ([]domain.VersionInfo, error) {
	extension, err := r.getExtension(ctx, id, baseFlags|FlagIncludeVersions)
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
			r.logFunc(fmt.Sprintf("parse version %q: %v", ver.Version, err))
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
		// Пропускаем версии у которых все варианты — pre-release
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
func (r *Registry) extensionQuery(ctx context.Context, searchReq searchRequest) (SearchResponse, error) {
	var lastErr error
	for attempt := range DefaultQueryRetries {
		if ctx.Err() != nil {
			return SearchResponse{}, fmt.Errorf("extension query: %w", ctx.Err())
		}

		queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
		resp, err := r.doExtensionQuery(queryCtx, searchReq)
		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err
		var qe *queryError
		if errors.As(err, &qe) && !qe.retryable {
			return SearchResponse{}, qe.err
		}
		r.logFunc(fmt.Sprintf("query attempt %d/%d failed: %v", attempt+1, DefaultQueryRetries, err))
	}
	return SearchResponse{}, lastErr
}

func (r *Registry) doExtensionQuery(ctx context.Context, searchRequest searchRequest) (SearchResponse, error) {
	body, err := json.Marshal(searchRequest)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/extensionquery", r.url), bytes.NewBuffer(body))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;api-version=7.1-preview.1")

	resp, err := r.client.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SearchResponse{}, &queryError{
			err:       fmt.Errorf("make search query: unexpected response status code %d", resp.StatusCode),
			retryable: resp.StatusCode >= 500,
		}
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}
	searchResponse := SearchResponse{}
	err = json.Unmarshal(bytes, &searchResponse)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
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

// Скачивает расширение из источника(ссылки) source
func (r *Registry) downloadFromSource(ctx context.Context, source string, onProgress domain.ProgressFunc) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("download from source: %w", err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download from source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download from source: unexpected response status code %d", resp.StatusCode)
	}

	reader := httputil.NewStallReader(httputil.NewProgressReader(resp.Body, resp.ContentLength, onProgress), r.sourceTimeout)
	return io.ReadAll(reader)
}

// Делает Head-запрос для получения размера расширения
func (r *Registry) getSize(ctx context.Context, sources []string) (int64, error) {
	for _, source := range sources {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, source, nil)
		if err != nil {
			r.logFunc(fmt.Sprintf("get size: %s", err))
			continue
		}
		resp, err := r.client.Do(req)
		if err != nil {
			r.logFunc(fmt.Sprintf("get size: %s", err))
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			r.logFunc(fmt.Sprintf("source %s unavailable: status %d", source, resp.StatusCode))
			continue
		}
		return resp.ContentLength, nil
	}
	return 0, fmt.Errorf("get size: %w", domain.ErrAllSourcesUnavailable)
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
