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
	url    string
	client *http.Client
}

func NewRegistry(url string, client *http.Client) *Registry {
	return &Registry{
		url:    url,
		client: client,
	}
}

func (marketplace *Registry) Search(ctx context.Context, query string, count int) ([]domain.Extension, error) {
	data := searchRequest{
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

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/extensionquery", marketplace.url), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json;api-version=7.1-preview.1")

	resp, err := marketplace.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search extensions: unexpected response status code %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}
	searchResponse := SearchResponse{}
	err = json.Unmarshal(bytes, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}

	if len(searchResponse.Results) == 0 {
		return []domain.Extension{}, nil
	}

	responseResult := searchResponse.Results[0]
	var result []domain.Extension
	for _, extension := range responseResult.Extensions {
		releaseVersion, found := findReleaseVersion(extension.Versions)
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

func (marketplace *Registry) GetLatestVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	return domain.Version{}, nil
}

func (marketplace *Registry) Download(ctx context.Context, id domain.ExtensionID, version domain.Version, onProgress domain.ProgressFunc) (io.ReadCloser, error) {
	return nil, nil
}

func findReleaseVersion(versions []Version) (Version, bool) {
	for _, v := range versions {
		if !isPreRelease(v) {
			return v, true
		}
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
