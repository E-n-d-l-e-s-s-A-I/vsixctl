package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
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
)

type Registry struct {
	url           string
	client        *http.Client
	platform      domain.Platform // Платформа на которой запущена утилита
	sourceTimeout time.Duration   // Таймаут на ответ источника при скачивании. По истечении таймаута переходим к следующему источнику
	logFunc       domain.LogFunc
}

const (
	DefaultURL               = "https://marketplace.visualstudio.com/_apis/public/gallery"
	DefaultMaxIdleConns      = 100
	DefaultMaxConnsPerHost   = 10
	DefaultIdleConnTimeout   = 90 * time.Second
	DefaultSHandshakeTimeout = 5 * time.Second
	DefaultTimeout           = 3 * time.Minute
)

func NewRegistry(url string, client *http.Client, platform domain.Platform, sourceTimeout time.Duration, logFunc domain.LogFunc) *Registry {
	if logFunc == nil {
		logFunc = func(string) {}
	}
	return &Registry{
		url:           url,
		client:        client,
		platform:      platform,
		sourceTimeout: sourceTimeout,
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
			ResponseHeaderTimeout: DefaultSHandshakeTimeout,
		},
		Timeout: DefaultTimeout,
	}
}

func (r *Registry) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	searchRequest := searchRequest{
		Filters: []searchFilter{
			{
				Criteria: []searchCriteria{
					{
						FilterType: TextSearch,
						Value:      query,
					},
				},
				PageNumber: 1,
				PageSize:   count,
				SortBy:     SortByRelevance,
				SortOrder:  SortOrderDefault,
			},
		},
		AssetTypes: []string{},
		Flags:      FlagIncludeVersions | FlagIncludeFiles | FlagIncludeVersionProps | FlagIncludeAssetUri | FlagIncludeStatistics,
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
		releaseVersion, found := findLatestReleaseVersion(extension.Versions, r.platform)
		if !found {
			continue
		}
		domainExt, err := r.marketplaceExtensionToDomain(extension, releaseVersion)
		if err != nil {
			continue
		}
		result = append(result, domainExt)
	}

	return result, nil
}

func (r *Registry) getExtension(ctx context.Context, id domain.ExtensionID) (Extension, error) {
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
		Flags:      FlagIncludeVersions | FlagIncludeFiles | FlagIncludeVersionProps | FlagIncludeAssetUri | FlagIncludeStatistics,
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

func (r *Registry) GetDownloadInfo(ctx context.Context, id domain.ExtensionID) (domain.Extension, domain.DownloadInfo, error) {
	extension, err := r.getExtension(ctx, id)
	if err != nil {
		return domain.Extension{}, domain.DownloadInfo{}, fmt.Errorf("get download info: %w", err)
	}
	releaseVersion, found := findLatestReleaseVersion(extension.Versions, r.platform)
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

// findLatestReleaseVersion находит последнюю релизную версию для платформы.
// Platform-specific версия имеет приоритет над универсальной.
func findLatestReleaseVersion(versions []Version, platform domain.Platform) (Version, bool) {
	var universalFallback *Version
	for _, v := range versions {
		if isPreRelease(v) {
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

func isPreRelease(v Version) bool {
	for _, p := range v.Properties {
		if p.Key == "Microsoft.VisualStudio.Code.PreRelease" && p.Value == "true" {
			return true
		}
	}
	return false
}

func (r *Registry) extensionQuery(ctx context.Context, searchRequest searchRequest) (SearchResponse, error) {
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
		return SearchResponse{}, fmt.Errorf("make search query: unexpected response status code %d", resp.StatusCode)
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
