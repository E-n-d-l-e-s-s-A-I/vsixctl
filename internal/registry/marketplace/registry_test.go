package marketplace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

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

			registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64, 5*time.Second)
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

			registry := NewRegistry(server.URL, server.Client(), testCase.platform, 5*time.Second)
			got, err := registry.GetLatestVersion(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"})

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && reflect.DeepEqual(got, testCase.wantVersionInfo) {
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

			registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64, 5*time.Second)
			versionInfo := domain.VersionInfo{
				Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
				Source:  server.URL,
			}
			noopProgress := func(downloaded, total int64) {}

			data, err := registry.Download(context.Background(), versionInfo, noopProgress)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr {
				if string(data) != testCase.wantBody {
					t.Errorf("got body %q, want %q", string(data), testCase.wantBody)
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

	registry := NewRegistry(server.URL, server.Client(), domain.LinuxX64, 5*time.Second)
	versionInfo := domain.VersionInfo{
		Version: domain.Version{Major: 1, Minor: 0, Patch: 0},
		Source:  server.URL,
	}

	onProgressCalls := 0
	onProgress := func(downloaded, total int64) {
		onProgressCalls += 1
	}

	data, err := registry.Download(context.Background(), versionInfo, onProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != body {
		t.Errorf("got body %q, want %q", string(data), body)
	}

	// Проверяем что колбэк был вызван нужное
	if onProgressCalls == 0 {
		t.Error("expected onProgress to be called at least once")
	}
}

func TestDownloadFallback(t *testing.T) {
	noopProgress := func(downloaded, total int64) {}

	t.Run("source_fails_fallback_succeeds", func(t *testing.T) {
		// Основной источник отдаёт 500, fallback отдаёт контент
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("vsix-data"))
		}))
		defer okServer.Close()

		registry := NewRegistry("", failServer.Client(), domain.LinuxX64, 5*time.Second)
		versionInfo := domain.VersionInfo{
			Version:         domain.Version{Major: 1, Minor: 0, Patch: 0},
			Source:          failServer.URL,
			FallbackSources: []string{okServer.URL},
		}

		data, err := registry.Download(context.Background(), versionInfo, noopProgress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != "vsix-data" {
			t.Errorf("got %q, want %q", string(data), "vsix-data")
		}
	})

	t.Run("all_sources_fail", func(t *testing.T) {
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer failServer.Close()

		registry := NewRegistry("", failServer.Client(), domain.LinuxX64, 5*time.Second)
		versionInfo := domain.VersionInfo{
			Version:         domain.Version{Major: 1, Minor: 0, Patch: 0},
			Source:          failServer.URL,
			FallbackSources: []string{failServer.URL},
		}

		_, err := registry.Download(context.Background(), versionInfo, noopProgress)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("stall_triggers_fallback", func(t *testing.T) {
		// Первый источник зависает после нескольких байт
		stallServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("partial"))
			w.(http.Flusher).Flush()
			// Зависаем - не отправляем остальные данные
			<-r.Context().Done()
		}))
		defer stallServer.Close()

		okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("complete-vsix"))
		}))
		defer okServer.Close()

		// Короткий таймаут чтобы stall сработал быстро
		registry := NewRegistry("", stallServer.Client(), domain.LinuxX64, 100*time.Millisecond)
		versionInfo := domain.VersionInfo{
			Version:         domain.Version{Major: 1, Minor: 0, Patch: 0},
			Source:          stallServer.URL,
			FallbackSources: []string{okServer.URL},
		}

		data, err := registry.Download(context.Background(), versionInfo, noopProgress)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != "complete-vsix" {
			t.Errorf("got %q, want %q", string(data), "complete-vsix")
		}
	})

	t.Run("context_cancelled_stops_fallback", func(t *testing.T) {
		// Оба источника отдают 500, но контекст уже отменён
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		registry := NewRegistry("", failServer.Client(), domain.LinuxX64, 5*time.Second)
		versionInfo := domain.VersionInfo{
			Version:         domain.Version{Major: 1, Minor: 0, Patch: 0},
			Source:          failServer.URL,
			FallbackSources: []string{failServer.URL},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := registry.Download(ctx, versionInfo, noopProgress)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
