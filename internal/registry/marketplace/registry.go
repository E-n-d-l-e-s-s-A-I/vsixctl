package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/httputil"
)

const VsixAssetPath = "/Microsoft.VisualStudio.Services.VSIXPackage"

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
	DefaultTimeout           = 10 * time.Minute
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
			MaxIdleConns:        DefaultMaxIdleConns,
			MaxConnsPerHost:     DefaultMaxConnsPerHost,
			IdleConnTimeout:     DefaultIdleConnTimeout,
			TLSHandshakeTimeout: DefaultSHandshakeTimeout,
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
		version, err := domain.ParseVersion(releaseVersion.Version)
		if err != nil {
			continue
		}

		result = append(result, domain.Extension{
			ID: domain.ExtensionID{
				Name:      extension.ExtensionName,
				Publisher: extension.Publisher.PublisherName,
			},
			Description: extension.ShortDescription,
			Version:     version,
		})
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

func (r *Registry) GetLatestVersion(ctx context.Context, id domain.ExtensionID) (domain.VersionInfo, error) {
	extension, err := r.getExtension(ctx, id)
	if err != nil {
		return domain.VersionInfo{}, fmt.Errorf("get latest version: %w", err)
	}
	if len(extension.Versions) < 1 {
		return domain.VersionInfo{}, fmt.Errorf("get latest version: %w", domain.ErrVersionNotFound)
	}

	lastReleaseVersion, ok := findLatestReleaseVersion(extension.Versions, r.platform)
	if !ok {
		return domain.VersionInfo{}, fmt.Errorf("get latest version: %w", domain.ErrVersionNotFound)
	}
	version, err := domain.ParseVersion(lastReleaseVersion.Version)
	if err != nil {
		return domain.VersionInfo{}, fmt.Errorf("get latest version: %w", err)
	}

	// Прямая ссылка на скачивание
	directUri := fmt.Sprintf("%s/publishers/%s/vsextensions/%s/%s/vspackage", r.url, id.Publisher, id.Name, version.String())
	if lastReleaseVersion.TargetPlatform != "" {
		directUri += fmt.Sprintf("?targetPlatform=%s", lastReleaseVersion.TargetPlatform)
	}

	return domain.VersionInfo{
		Version:         version,
		Source:          lastReleaseVersion.AssetUri + VsixAssetPath,
		FallbackSources: []string{lastReleaseVersion.FallbackAssetUri + VsixAssetPath, directUri},
	}, nil
}

// Скачивание vsix-пакета, учитывает разные источники
// Если источник недоступен переходит к следующему
func (r *Registry) Download(ctx context.Context, versionInfo domain.VersionInfo, onProgress domain.ProgressFunc) ([]byte, error) {
	// Формирование списка источников
	sources := append([]string{versionInfo.Source}, versionInfo.FallbackSources...)

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
