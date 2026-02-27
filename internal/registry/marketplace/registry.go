package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type Registry struct {
	url      string
	client   *http.Client
	platform domain.Platform
}

func NewRegistry(url string, client *http.Client, platform domain.Platform) *Registry {
	return &Registry{
		url:      url,
		client:   client,
		platform: platform,
	}
}

func (marketplace *Registry) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
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
	searchResponse, err := marketplace.extensionQuery(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}
	if len(searchResponse.Results) == 0 {
		return []domain.Extension{}, nil
	}

	responseResult := searchResponse.Results[0]
	var result []domain.Extension
	for _, extension := range responseResult.Extensions {
		releaseVersion, found := findLatestReleaseVersion(extension.Versions, marketplace.platform)
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

func (marketplace *Registry) getExtension(ctx context.Context, id domain.ExtensionID) (Extension, error) {
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
	searchResponse, err := marketplace.extensionQuery(ctx, searchRequest)
	if err != nil {
		return Extension{}, fmt.Errorf("get extension: %w", err)
	}
	if len(searchResponse.Results) < 1 {
		return Extension{}, fmt.Errorf("get extension: extension not found")
	}
	if len(searchResponse.Results[0].Extensions) < 1 {
		return Extension{}, fmt.Errorf("get extension: extension not found")
	}
	return searchResponse.Results[0].Extensions[0], nil
}

func (marketplace *Registry) GetLatestVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	extension, err := marketplace.getExtension(ctx, id)
	if err != nil {
		return domain.Version{}, fmt.Errorf("get latest version: %w", err)
	}
	if len(extension.Versions) < 1 {
		return domain.Version{}, fmt.Errorf("get latest version: versions not found")
	}

	lastReleaseVersion, ok := findLatestReleaseVersion(extension.Versions, marketplace.platform)
	if !ok {
		return domain.Version{}, fmt.Errorf("get latest version: latest release version not found")
	}
	version, err := domain.ParseVersion(lastReleaseVersion.Version)
	if err != nil {
		return domain.Version{}, fmt.Errorf("get latest version: %w", err)
	}

	return version, nil
}

func (marketplace *Registry) Download(ctx context.Context, id domain.ExtensionID, version domain.Version, onProgress domain.ProgressFunc) (io.ReadCloser, error) {
	return nil, nil
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

func (marketplace *Registry) extensionQuery(ctx context.Context, searchRequest searchRequest) (SearchResponse, error) {
	body, err := json.Marshal(searchRequest)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/extensionquery", marketplace.url), bytes.NewBuffer(body))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("make search query: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;api-version=7.1-preview.1")

	resp, err := marketplace.client.Do(req)
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
