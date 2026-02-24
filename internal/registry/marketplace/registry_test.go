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
											"extensionName": "Go",
											"shortDescription": "Go support",
											"publisher": {
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
					ID: domain.ExtensionID{
						Name:      "Go",
						Publisher: "golang",
					},
					Description: "Go support",
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
											"extensionName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherName": "golang"
											},
											"versions": [
												{
											        "version": "1.0.0"
												}
											]
										},
										{
											"extensionName": "Go lint",
											"shortDescription": "Go lint",
											"publisher": {
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "1.0.0"
												}
											]
										},
										{
											"extensionName": "Go fmt",
											"shortDescription": "Go fmt",
											"publisher": {
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
					ID: domain.ExtensionID{
						Name:      "Go",
						Publisher: "golang",
					},
					Description: "Go support",
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
				{
					ID: domain.ExtensionID{
						Name:      "Go lint",
						Publisher: "golang",
					},
					Description: "Go lint",
					Version: domain.Version{
						Major: 1,
						Minor: 0,
						Patch: 0,
					},
				},
				{
					ID: domain.ExtensionID{
						Name:      "Go fmt",
						Publisher: "golang",
					},
					Description: "Go fmt",
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
			name:       "skips_invalid_versions",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionName": "Bad",
											"shortDescription": "Bad version",
											"publisher": {
												"publisherName": "test"
											},
											"versions": [
												{
													"version": "a.b.c"
												}
											]
										},
										{
											"extensionName": "Good",
											"shortDescription": "Good version",
											"publisher": {
												"publisherName": "test"
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
					ID: domain.ExtensionID{
						Name:      "Good",
						Publisher: "test",
					},
					Description: "Good version",
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
			name:       "skips_prerelease_version",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "2.0.0",
													"properties": [
														{
															"key": "Microsoft.VisualStudio.Code.PreRelease",
															"value": "true"
														}
													]
												},
												{
													"version": "1.5.0"
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
					ID: domain.ExtensionID{
						Name:      "Go",
						Publisher: "golang",
					},
					Description: "Go support",
					Version: domain.Version{
						Major: 1,
						Minor: 5,
						Patch: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "all_prerelease_versions",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionName": "Go",
											"shortDescription": "Go support",
											"publisher": {
												"publisherName": "golang"
											},
											"versions": [
												{
													"version": "2.0.0",
													"properties": [
														{
															"key": "Microsoft.VisualStudio.Code.PreRelease",
															"value": "true"
														}
													]
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
			wantErr:     false,
		},
		{
			name:       "skips_without_versions",
			statusCode: http.StatusOK,
			query:      "go",
			response: `{
							"results": [
							    {
									"extensions": [
										{
											"extensionName": "NoVersions",
											"shortDescription": "No versions",
											"publisher": {
												"publisherName": "test"
											},
											"versions": []
										},
										{
											"extensionName": "Good",
											"shortDescription": "Good version",
											"publisher": {
												"publisherName": "test"
											},
											"versions": [
												{
													"version": "2.0.0"
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
					ID: domain.ExtensionID{
						Name:      "Good",
						Publisher: "test",
					},
					Description: "Good version",
					Version: domain.Version{
						Major: 2,
						Minor: 0,
						Patch: 0,
					},
				},
			},
			wantErr: false,
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
