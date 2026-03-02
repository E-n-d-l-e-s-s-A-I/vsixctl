package marketplace

import (
	"context"
	"io"
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

			registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64)
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

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		statusCode      int
		platform        domain.Platform
		wantVersionInfo domain.VersionInfo
		wantErr         bool
	}{
		{
			name:       "universal_extension",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "go",
						"publisher": {"publisherName": "golang"},
						"versions": [
							{"version": "1.5.0", "assetUri": "https://cdn.example.com/go/1.5.0"},
							{"version": "1.4.0", "assetUri": "https://cdn.example.com/go/1.4.0"}
						]
					}]
				}]
			}`,
			wantVersionInfo: domain.VersionInfo{
				Version: domain.Version{Major: 1, Minor: 5, Patch: 0},
				Source:  "https://cdn.example.com/go/1.5.0",
			},
		},
		{
			name:       "platform_specific",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "debugpy",
						"publisher": {"publisherName": "ms-python"},
						"versions": [
							{"version": "2.0.0", "targetPlatform": "linux-x64", "assetUri": "https://cdn.example.com/debugpy/2.0.0/linux-x64"},
							{"version": "2.0.0", "targetPlatform": "darwin-arm64", "assetUri": "https://cdn.example.com/debugpy/2.0.0/darwin-arm64"},
							{"version": "1.0.0", "assetUri": "https://cdn.example.com/debugpy/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersionInfo: domain.VersionInfo{
				Version: domain.Version{Major: 2, Minor: 0, Patch: 0},
				Source:  "https://cdn.example.com/debugpy/2.0.0/linux-x64",
			},
		},
		{
			name:       "return_latest_version",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "debugpy",
						"publisher": {"publisherName": "ms-python"},
						"versions": [
							{"version": "2.0.0", "targetPlatform": "linux-x64", "assetUri": "https://cdn.example.com/debugpy/2.0.0"},
							{"version": "1.0.0", "targetPlatform": "linux-x64", "assetUri": "https://cdn.example.com/debugpy/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersionInfo: domain.VersionInfo{
				Version: domain.Version{Major: 2, Minor: 0, Patch: 0},
				Source:  "https://cdn.example.com/debugpy/2.0.0",
			},
		},
		{
			name:       "extension_not_found",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response:   `{"results": [{"extensions": []}]}`,
			wantErr:    true,
		},
		{
			name:       "empty_results",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response:   `{"results": []}`,
			wantErr:    true,
		},
		{
			name:       "no_suitable_version",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "1.0.0", "targetPlatform": "darwin-arm64"}
						]
					}]
				}]
			}`,
			wantErr: true,
		},
		{
			name:       "no_versions",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": []
					}]
				}]
			}`,
			wantErr: true,
		},
		{
			name:       "server_error",
			statusCode: http.StatusInternalServerError,
			platform:   domain.LinuxX64,
			response:   "",
			wantErr:    true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.statusCode)
				w.Write([]byte(testCase.response))
			}))
			defer server.Close()

			registry := NewRegistry(server.URL, server.Client(), testCase.platform)
			got, err := registry.GetLatestVersion(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"})

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && got != testCase.wantVersionInfo {
				t.Errorf("got %+v, want %+v", got, testCase.wantVersionInfo)
			}
		})
	}
}

func TestDownload(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantBody   string
		wantErr    bool
	}{
		{
			name:       "successful_download",
			statusCode: http.StatusOK,
			body:       "fake-vsix-content",
			wantBody:   "fake-vsix-content",
		},
		{
			name:       "server_error",
			statusCode: http.StatusInternalServerError,
			body:       "",
			wantErr:    true,
		},
		{
			name:       "not_found",
			statusCode: http.StatusNotFound,
			body:       "",
			wantErr:    true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.statusCode)
				w.Write([]byte(testCase.body))
			}))
			defer server.Close()

			registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64)
			versionInfo := domain.VersionInfo{
				Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
				Source:  server.URL,
			}
			noopProgress := func(downloaded, total int64) {}

			reader, err := registry.Download(context.Background(), versionInfo, noopProgress)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr {
				defer reader.Close()
				got, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}
				if string(got) != testCase.wantBody {
					t.Errorf("got body %q, want %q", string(got), testCase.wantBody)
				}
			}
		})
	}
}

func TestDownloadProgress(t *testing.T) {
	body := "abcdefghij" // 10 байт
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64)
	versionInfo := domain.VersionInfo{
		Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
		Source:  server.URL,
	}

	onProgressCalls := 0
	onProgress := func(downloaded, total int64) {
		onProgressCalls += 1
	}

	reader, err := registry.Download(context.Background(), versionInfo, onProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	// Читаем всё содержимое по 2 байта
	batch := make([]byte, 2)
	for {
		n, err := reader.Read(batch)
		if err != nil || n == 0 {
			break
		}
	}

	// Проверяем кол-во вызовов колбэка
	if onProgressCalls != 5 {
		t.Errorf("got onProgressCalls = %d, want 5", onProgressCalls)
	}
}
