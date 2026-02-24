package marketplace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestSearch(t *testing.T) {
	tests := []struct {
		name        string
		response    string // JSON который вернёт фейковый сервер
		statusCode  int
		query       string
		searchCount int
		wantResults []domain.Extension
		wantErr     bool
	}{
		{
			name:       "single_result",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
												{
											        "version": "1.0.0"
												}
											]
										}
									],
									"resultMetadata": []
								}
							]
						}`,
			searchCount: 10,
			wantResults: []domain.Extension{
				{
					Name:        "Go",
					Description: "Go support",
					Publisher: domain.Publisher{
						Name: "golang",
					},
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "multiple_results",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
												{
											        "version": "1.0.0"
												}
											]
										},
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go lint",
											"shortDescription": "Go lint",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "1.0.0"
												}
											]
										},
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go fmt",
											"shortDescription": "Go fmt",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "1.0.0"
												}
											]
										}
									],
									"resultMetadata": []
								}
							]
						}`,
			searchCount: 10,
			wantResults: []domain.Extension{
				{
					Name:        "Go",
					Description: "Go support",
					Publisher: domain.Publisher{
						Name: "golang",
					},
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
				{
					Name:        "Go lint",
					Description: "Go lint",
					Publisher: domain.Publisher{
						Name: "golang",
					},
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
				{
					Name:        "Go fmt",
					Description: "Go fmt",
					Publisher: domain.Publisher{
						Name: "golang",
					},
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "empty_results",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": []
						}`,
			searchCount: 10,
			wantResults: []domain.Extension{},
			wantErr:     false,
		},
		{
			name:        "server_error",
			statusCode:  http.StatusInternalServerError,
			query:       "go",
			response:    "",
			searchCount: 10,
			wantResults: nil,
			wantErr:     true,
		},
		{
			name:        "invalid_json",
			statusCode:  http.StatusOK,
			query:       "go",
			response:    `{"invalidJson"}`,
			searchCount: 10,
			wantResults: nil,
			wantErr:     true,
		},
		{
			name:       "invalid_uuid_in_response",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionId": "1",
											"displayName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherId": "1",
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "1.0.0"
												}
											]
										}
									],
									"resultMetadata": []
								}
							]
						}`,
			searchCount: 10,
			wantResults: nil,
			wantErr:     true,
		},
		{
			name:       "invalid_versions",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "a.b.c"
												}
											]
										}
									],
									"resultMetadata": []
								}
							]
						}`,
			searchCount: 10,
			wantResults: nil,
			wantErr:     true,
		},
		{
			name:       "without_versions",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionId": "00000000-0000-0000-0000-000000000000",
											"displayName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherId": "00000000-0000-0000-0000-000000000000",
												"publisherName": "golang"
											},
											"versions": [
											]
										}
									],
									"resultMetadata": []
								}
							]
						}`,
			searchCount: 10,
			wantResults: nil,
			wantErr:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.statusCode)
				w.Write([]byte(testCase.response))
			}))
			defer server.Close()

			registry := NewRegistry(server.URL, server.Client())
			results, err := registry.Search(context.Background(), testCase.query, testCase.searchCount)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && !reflect.DeepEqual(results, testCase.wantResults) {
				t.Errorf("got %+v, want %+v", results, testCase.wantResults)
			}
		})
	}
}
