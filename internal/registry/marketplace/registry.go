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
	URL    string
	client *http.Client
}

func NewRegistry(url string, client *http.Client) *Registry {
	return &Registry{
		URL:    url,
		client: client,
	}
}

func (marketplace *Registry) Search(ctx context.Context, query string) ([]domain.Extension, error) {
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
				PageSize:   1,
				SortBy:     SortByRelevance,
				SortOrder:  SortOrderDefault,
			},
		},
		AssetTypes: []string{},
		Flags:      FlagIncludeVersions | FlagIncludeFiles | FlagIncludeVersionProps | FlagIncludeAssetUri | FlagIncludeStatistics | FlagIncludeLatestOnly,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("search extensions: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/extensionquery", marketplace.URL), bytes.NewBuffer(body))
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
	result := make([]domain.Extension, len(responseResult.Extensions))
	for i, extension := range responseResult.Extensions {
		domainExtension := domain.Extension{
			ID: extension.ExtensionId,
			Publisher: domain.Publisher{
				ID:   extension.Publisher.PublisherId,
				Name: extension.Publisher.PublisherName,
			},
			Name:        extension.DisplayName,
			Description: extension.ShortDescription,
		}
		result[i] = domainExtension
	}

	return result, nil
}

func (marketplace *Registry) GetLatestVersion(ctx context.Context, id domain.ExtensionID) (domain.Version, error) {
	return domain.Version{}, nil
}

func (marketplace *Registry) Download(ctx context.Context, id domain.ExtensionID, version domain.Version, onProgress domain.ProgressFunc) (io.ReadCloser, error) {
	return nil, nil
}
