package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

var vscodeVer = domain.Version{
	Major: 1,
	Minor: 107,
	Patch: 1,
}

func TestSearch(t *testing.T) {
	tests := []struct {
		name        string
		response    string // JSON который вернёт фейковый сервер
		statusCode  int
		query       domain.SearchQuery
		wantResults []domain.Extension
		wantErr     bool
	}{
		{
			name:       "single_result",
			statusCode: http.StatusOK,
			query:      domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText},
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
			wantResults: []domain.Extension{
				{
					ID: domain.ExtensionID{
						Name:      "Go",
						Publisher: "golang",
					},
					Description: "Go support",
				},
			},
			wantErr: false,
		},
		{
			name:       "multiple_results",
			statusCode: http.StatusOK,
			query:      domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText},
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
			wantResults: []domain.Extension{
				{
					ID: domain.ExtensionID{
						Name:      "Go",
						Publisher: "golang",
					},
					Description: "Go support",
				},
				{
					ID: domain.ExtensionID{
						Name:      "Go lint",
						Publisher: "golang",
					},
					Description: "Go lint",
				},
				{
					ID: domain.ExtensionID{
						Name:      "Go fmt",
						Publisher: "golang",
					},
					Description: "Go fmt",
				},
			},
			wantErr: false,
		},
		{
			name:       "empty_results",
			statusCode: http.StatusOK,
			query:      domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText},
			response: `{
							"results": []
						}`,
			wantResults: []domain.Extension{},
			wantErr:     false,
		},
		{
			name:        "server_error",
			statusCode:  http.StatusInternalServerError,
			query:       domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText},
			response:    "",
			wantResults: nil,
			wantErr:     true,
		},
		{
			name:        "invalid_json",
			statusCode:  http.StatusOK,
			query:       domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText},
			response:    `{"invalidJson"}`,
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

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
			results, err := registry.Search(context.Background(), testCase.query)

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

	t.Run("count_passed_as_page_size", func(t *testing.T) {
		var gotPageSize int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req searchRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Filters) > 0 {
				gotPageSize = req.Filters[0].PageSize
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"results": []}`))
		}))
		defer server.Close()

		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
		registry.Search(context.Background(), domain.SearchQuery{Query: "go", Limit: 25, Type: domain.SearchByText})

		if gotPageSize != 25 {
			t.Errorf("pageSize: got %d, want 25", gotPageSize)
		}
	})

	t.Run("search_type_mapped_to_filter_type", func(t *testing.T) {
		tests := []struct {
			name           string
			searchType     domain.SearchType
			wantFilterType int
		}{
			{
				name:           "text_search",
				searchType:     domain.SearchByText,
				wantFilterType: TextSearch,
			},
			{
				name:           "id_search",
				searchType:     domain.SearchByID,
				wantFilterType: ExtensionIdSearch,
			},
			{
				name:           "name_search",
				searchType:     domain.SearchByName,
				wantFilterType: DisplayNameSearch,
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var gotFilterType int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req searchRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Filters) > 0 && len(req.Filters[0].Criteria) > 0 {
						gotFilterType = req.Filters[0].Criteria[0].FilterType
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"results": []}`))
				}))
				defer server.Close()

				registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
				registry.Search(context.Background(), domain.SearchQuery{Query: "test", Limit: 10, Type: testCase.searchType})

				if gotFilterType != testCase.wantFilterType {
					t.Errorf("filterType: got %d, want %d", gotFilterType, testCase.wantFilterType)
				}
			})
		}
	})
}

func TestDownloadInfo(t *testing.T) {
	const testSize int64 = 12345

	tests := []struct {
		name        string
		response    string // {{BASE_URL}} заменяется на server.URL в рантайме
		statusCode  int
		platform    domain.Platform
		wantVersion domain.Version
		wantSource  string // {{BASE_URL}} заменяется на server.URL
		wantSize    int64
		wantPackIDs []domain.ExtensionID
		wantDepIDs  []domain.ExtensionID
		wantErr     bool
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
							{"version": "1.5.0", "assetUri": "{{BASE_URL}}/go/1.5.0"},
							{"version": "1.4.0", "assetUri": "{{BASE_URL}}/go/1.4.0"}
						]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 1, Minor: 5, Patch: 0},
			wantSource:  "{{BASE_URL}}/go/1.5.0" + VsixAssetPath,
			wantSize:    testSize,
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
							{"version": "2.0.0", "targetPlatform": "linux-x64", "assetUri": "{{BASE_URL}}/debugpy/2.0.0/linux-x64"},
							{"version": "2.0.0", "targetPlatform": "darwin-arm64", "assetUri": "{{BASE_URL}}/debugpy/2.0.0/darwin-arm64"},
							{"version": "1.0.0", "assetUri": "{{BASE_URL}}/debugpy/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 2, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/debugpy/2.0.0/linux-x64" + VsixAssetPath,
			wantSize:    testSize,
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
							{"version": "2.0.0", "targetPlatform": "linux-x64", "assetUri": "{{BASE_URL}}/debugpy/2.0.0"},
							{"version": "1.0.0", "targetPlatform": "linux-x64", "assetUri": "{{BASE_URL}}/debugpy/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 2, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/debugpy/2.0.0" + VsixAssetPath,
			wantSize:    testSize,
		},
		{
			name:       "with_extension_pack",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "python",
						"publisher": {"publisherName": "ms-python"},
						"versions": [{
							"version": "1.0.0",
							"assetUri": "{{BASE_URL}}/python/1.0.0",
							"properties": [
								{"key": "Microsoft.VisualStudio.Code.ExtensionPack", "value": "ms-python.debugpy,ms-python.vscode-pylance"}
							]
						}]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 1, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/python/1.0.0" + VsixAssetPath,
			wantSize:    testSize,
			wantPackIDs: []domain.ExtensionID{
				{Publisher: "ms-python", Name: "debugpy"},
				{Publisher: "ms-python", Name: "vscode-pylance"},
			},
		},
		{
			name:       "with_dependencies",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "pylance",
						"publisher": {"publisherName": "ms-python"},
						"versions": [{
							"version": "3.0.0",
							"assetUri": "{{BASE_URL}}/pylance/3.0.0",
							"properties": [
								{"key": "Microsoft.VisualStudio.Code.ExtensionDependencies", "value": "ms-python.python"}
							]
						}]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 3, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/pylance/3.0.0" + VsixAssetPath,
			wantSize:    testSize,
			wantDepIDs:  []domain.ExtensionID{{Publisher: "ms-python", Name: "python"}},
		},
		{
			name:       "with_pack_and_dependencies",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "python",
						"publisher": {"publisherName": "ms-python"},
						"versions": [{
							"version": "1.0.0",
							"assetUri": "{{BASE_URL}}/python/1.0.0",
							"properties": [
								{"key": "Microsoft.VisualStudio.Code.ExtensionPack", "value": "ms-python.debugpy"},
								{"key": "Microsoft.VisualStudio.Code.ExtensionDependencies", "value": "ms-python.vscode-pylance"}
							]
						}]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 1, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/python/1.0.0" + VsixAssetPath,
			wantSize:    testSize,
			wantPackIDs: []domain.ExtensionID{{Publisher: "ms-python", Name: "debugpy"}},
			wantDepIDs:  []domain.ExtensionID{{Publisher: "ms-python", Name: "vscode-pylance"}},
		},
		{
			name:       "invalid_id_in_extension_pack",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "pack",
						"publisher": {"publisherName": "test"},
						"versions": [{
							"version": "1.0.0",
							"assetUri": "{{BASE_URL}}/pack/1.0.0",
							"properties": [
								{"key": "Microsoft.VisualStudio.Code.ExtensionPack", "value": "invalid-format"}
							]
						}]
					}]
				}]
			}`,
			wantErr: true,
		},
		{
			name:       "no_pack_properties",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "simple",
						"publisher": {"publisherName": "test"},
						"versions": [{
							"version": "1.0.0",
							"assetUri": "{{BASE_URL}}/simple/1.0.0"
						}]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 1, Minor: 0, Patch: 0},
			wantSource:  "{{BASE_URL}}/simple/1.0.0" + VsixAssetPath,
			wantSize:    testSize,
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
			var serverURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// HEAD-запросы для getSize
				if r.Method == http.MethodHead {
					w.Header().Set("Content-Length", "12345")
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(testCase.statusCode)
				response := strings.ReplaceAll(testCase.response, "{{BASE_URL}}", serverURL)
				w.Write([]byte(response))
			}))
			defer server.Close()
			serverURL = server.URL

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, testCase.platform, 5*time.Second, 15*time.Second, nil)
			ext, got, err := registry.GetDownloadInfo(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"}, nil)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.wantErr {
				return
			}

			wantSource := strings.ReplaceAll(testCase.wantSource, "{{BASE_URL}}", serverURL)
			if got.Version != testCase.wantVersion {
				t.Errorf("Version: got %+v, want %+v", got.Version, testCase.wantVersion)
			}
			if got.Source != wantSource {
				t.Errorf("Source: got %q, want %q", got.Source, wantSource)
			}
			if got.Size != testCase.wantSize {
				t.Errorf("Size: got %d, want %d", got.Size, testCase.wantSize)
			}
			if !reflect.DeepEqual(ext.ExtensionPack, testCase.wantPackIDs) {
				t.Errorf("ExtensionPack: got %+v, want %+v", ext.ExtensionPack, testCase.wantPackIDs)
			}
			if !reflect.DeepEqual(ext.Dependencies, testCase.wantDepIDs) {
				t.Errorf("Dependencies: got %+v, want %+v", ext.Dependencies, testCase.wantDepIDs)
			}
		})
	}
}

func TestDownloadInfoLatestOnlyOptimization(t *testing.T) {
	tests := []struct {
		name           string
		version        *domain.Version
		responses      []string // ответы на последовательные POST-запросы, {{BASE_URL}} заменяется в рантайме
		wantQueryCount int
		wantFlags      []int
	}{
		{
			name: "latest_compatible_single_query",
			responses: []string{
				`{
					"results": [{
						"extensions": [{
							"extensionName": "ext",
							"publisher": {"publisherName": "test"},
							"versions": [{
								"version": "2.0.0",
								"assetUri": "{{BASE_URL}}/ext/2.0.0"
							}]
						}]
					}]
				}`,
			},
			wantQueryCount: 1,
			wantFlags:      []int{baseFlags | FlagIncludeVersions | FlagIncludeLatestOnly},
		},
		{
			name: "latest_incompatible_fallback",
			responses: []string{
				// Первый запрос (LatestOnly) — версия несовместима по платформе
				`{
					"results": [{
						"extensions": [{
							"extensionName": "ext",
							"publisher": {"publisherName": "test"},
							"versions": [{
								"version": "2.0.0",
								"targetPlatform": "darwin-arm64",
								"assetUri": "{{BASE_URL}}/ext/2.0.0"
							}]
						}]
					}]
				}`,
				// Второй запрос (все версии) — есть совместимая старая версия
				`{
					"results": [{
						"extensions": [{
							"extensionName": "ext",
							"publisher": {"publisherName": "test"},
							"versions": [
								{"version": "2.0.0", "targetPlatform": "darwin-arm64", "assetUri": "{{BASE_URL}}/ext/2.0.0"},
								{"version": "1.0.0", "assetUri": "{{BASE_URL}}/ext/1.0.0"}
							]
						}]
					}]
				}`,
			},
			wantQueryCount: 2,
			wantFlags:      []int{baseFlags | FlagIncludeVersions | FlagIncludeLatestOnly, baseFlags | FlagIncludeVersions},
		},
		{
			name:    "specific_version_all_versions",
			version: &domain.Version{Major: 1, Minor: 0, Patch: 0},
			responses: []string{
				`{
					"results": [{
						"extensions": [{
							"extensionName": "ext",
							"publisher": {"publisherName": "test"},
							"versions": [
								{"version": "2.0.0", "assetUri": "{{BASE_URL}}/ext/2.0.0"},
								{"version": "1.0.0", "assetUri": "{{BASE_URL}}/ext/1.0.0"}
							]
						}]
					}]
				}`,
			},
			wantQueryCount: 1,
			wantFlags:      []int{baseFlags | FlagIncludeVersions},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				queryCount int
				gotFlags   []int
				serverURL  string
			)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.Header().Set("Content-Length", "12345")
					w.WriteHeader(http.StatusOK)
					return
				}
				var req searchRequest
				json.NewDecoder(r.Body).Decode(&req)
				gotFlags = append(gotFlags, req.Flags)

				idx := queryCount
				queryCount++
				if idx < len(testCase.responses) {
					response := strings.ReplaceAll(testCase.responses[idx], "{{BASE_URL}}", serverURL)
					w.Write([]byte(response))
				}
			}))
			defer server.Close()
			serverURL = server.URL

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
			registry.GetDownloadInfo(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"}, testCase.version)

			if queryCount != testCase.wantQueryCount {
				t.Errorf("query count: got %d, want %d", queryCount, testCase.wantQueryCount)
			}
			if !reflect.DeepEqual(gotFlags, testCase.wantFlags) {
				t.Errorf("flags: got %v, want %v", gotFlags, testCase.wantFlags)
			}
		})
	}
}

func TestParseExtensionIDs(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []domain.ExtensionID
		wantErr bool
	}{
		{
			name: "single_id",
			raw:  "ms-python.python",
			want: []domain.ExtensionID{{Publisher: "ms-python", Name: "python"}},
		},
		{
			name: "multiple_ids",
			raw:  "ms-python.python,golang.go,redhat.java",
			want: []domain.ExtensionID{
				{Publisher: "ms-python", Name: "python"},
				{Publisher: "golang", Name: "go"},
				{Publisher: "redhat", Name: "java"},
			},
		},
		{
			name: "empty_string",
			raw:  "",
			want: nil,
		},
		{
			name: "trailing_comma",
			raw:  "ms-python.python,",
			want: []domain.ExtensionID{{Publisher: "ms-python", Name: "python"}},
		},
		{
			name:    "invalid_id",
			raw:     "invalid-format",
			wantErr: true,
		},
		{
			name:    "invalid_id_among_valid",
			raw:     "ms-python.python,bad,golang.go",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseExtensionIDs(testCase.raw)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestFindProperty(t *testing.T) {
	properties := []Property{
		{Key: "Microsoft.VisualStudio.Code.ExtensionPack", Value: "ms-python.python,golang.go"},
		{Key: "Microsoft.VisualStudio.Code.ExtensionDependencies", Value: "redhat.java"},
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "existing_key",
			key:  "Microsoft.VisualStudio.Code.ExtensionPack",
			want: "ms-python.python,golang.go",
		},
		{
			name: "another_existing_key",
			key:  "Microsoft.VisualStudio.Code.ExtensionDependencies",
			want: "redhat.java",
		},
		{
			name: "missing_key",
			key:  "NonExistent",
			want: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := findProperty(properties, testCase.key)
			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestIsEngineCompatible(t *testing.T) {
	tests := []struct {
		name      string
		vscodeVer domain.Version
		engine    string
		want      bool
	}{
		{
			name:      "wildcard",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "*",
			want:      true,
		},
		{
			name:      "empty_string",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "",
			want:      true,
		},
		{
			name:      "caret_compatible",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "^1.80.0",
			want:      true,
		},
		{
			name:      "caret_exact_match",
			vscodeVer: domain.Version{Major: 1, Minor: 80, Patch: 0},
			engine:    "^1.80.0",
			want:      true,
		},
		{
			name:      "caret_too_old",
			vscodeVer: domain.Version{Major: 1, Minor: 70, Patch: 0},
			engine:    "^1.80.0",
			want:      false,
		},
		{
			name:      "gte_compatible",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    ">=1.80.0",
			want:      true,
		},
		{
			name:      "gte_exact_match",
			vscodeVer: domain.Version{Major: 1, Minor: 80, Patch: 0},
			engine:    ">=1.80.0",
			want:      true,
		},
		{
			name:      "tilde_compatible",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "~1.80.0",
			want:      true,
		},
		{
			name:      "major_version_too_low",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "^2.0.0",
			want:      false,
		},
		{
			name:      "patch_level_check",
			vscodeVer: domain.Version{Major: 1, Minor: 80, Patch: 1},
			engine:    "^1.80.2",
			want:      false,
		},
		{
			name:      "invalid_engine_value",
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			engine:    "invalid",
			want:      false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := isEngineCompatible(testCase.vscodeVer, testCase.engine)
			if got != testCase.want {
				t.Errorf("isEngineCompatible(%v, %q) = %v, want %v", testCase.vscodeVer, testCase.engine, got, testCase.want)
			}
		})
	}
}

func TestFindLatestSupportedVersion(t *testing.T) {
	tests := []struct {
		name        string
		versions    []Version
		vscodeVer   domain.Version
		platform    domain.Platform
		wantVersion string
		wantFound   bool
	}{
		{
			name: "compatible_platform_specific",
			versions: []Version{
				{Version: "2.0.0", TargetPlatform: "linux-x64", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.70.0"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "2.0.0",
			wantFound:   true,
		},
		{
			name: "platform_specific_priority_over_universal",
			versions: []Version{
				{Version: "3.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "2.0.0", TargetPlatform: "linux-x64", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "2.0.0",
			wantFound:   true,
		},
		{
			name: "falls_back_to_universal",
			versions: []Version{
				{Version: "2.0.0", TargetPlatform: "darwin-arm64", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.70.0"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "1.0.0",
			wantFound:   true,
		},
		{
			name: "skips_incompatible_engine_returns_older",
			versions: []Version{
				{Version: "3.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.100.0"}}},
				{Version: "2.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "2.0.0",
			wantFound:   true,
		},
		{
			name: "all_versions_incompatible",
			versions: []Version{
				{Version: "3.0.0", Properties: []Property{{Key: EngineProperty, Value: "^2.0.0"}}},
				{Version: "2.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.100.0"}}},
			},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
		{
			name: "no_engine_property_treated_as_compatible",
			versions: []Version{
				{Version: "1.0.0"},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "1.0.0",
			wantFound:   true,
		},
		{
			name: "skips_prerelease",
			versions: []Version{
				{Version: "3.0.0", Properties: []Property{{Key: "Microsoft.VisualStudio.Code.PreRelease", Value: "true"}}},
				{Version: "2.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "2.0.0",
			wantFound:   true,
		},
		{
			name:      "empty_versions",
			versions:  []Version{},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
		{
			name: "wildcard_engine",
			versions: []Version{
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "*"}}},
			},
			vscodeVer:   domain.Version{Major: 1, Minor: 50, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "1.0.0",
			wantFound:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, found := findLatestSupportedVersion(testCase.versions, testCase.vscodeVer, testCase.platform)
			if found != testCase.wantFound {
				t.Fatalf("found = %v, want %v", found, testCase.wantFound)
			}
			if found && got.Version != testCase.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, testCase.wantVersion)
			}
		})
	}
}

func TestFindSpecificSupportedVersion(t *testing.T) {
	tests := []struct {
		name               string
		versions           []Version
		version            domain.Version
		vscodeVer          domain.Version
		platform           domain.Platform
		wantVersion        string
		wantTargetPlatform string
		wantFound          bool
	}{
		{
			name: "exact_match_platform_specific",
			versions: []Version{
				{Version: "2.0.0", TargetPlatform: "linux-x64", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "1.0.0", TargetPlatform: "linux-x64", Properties: []Property{{Key: EngineProperty, Value: "^1.70.0"}}},
			},
			version:            domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer:          domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:           domain.LinuxX64,
			wantVersion:        "1.0.0",
			wantTargetPlatform: "linux-x64",
			wantFound:          true,
		},
		{
			name: "exact_match_universal",
			versions: []Version{
				{Version: "2.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.70.0"}}},
			},
			version:     domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer:   domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:    domain.LinuxX64,
			wantVersion: "1.0.0",
			wantFound:   true,
		},
		{
			name: "platform_specific_priority_over_universal",
			versions: []Version{
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
				{Version: "1.0.0", TargetPlatform: "linux-x64", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
			},
			version:            domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer:          domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:           domain.LinuxX64,
			wantVersion:        "1.0.0",
			wantTargetPlatform: "linux-x64",
			wantFound:          true,
		},
		{
			name: "version_exists_but_engine_incompatible",
			versions: []Version{
				{Version: "1.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.100.0"}}},
			},
			version:   domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
		{
			name: "version_exists_but_prerelease",
			versions: []Version{
				{Version: "1.0.0", Properties: []Property{
					{Key: EngineProperty, Value: "^1.80.0"},
					{Key: "Microsoft.VisualStudio.Code.PreRelease", Value: "true"},
				}},
			},
			version:   domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
		{
			name: "version_not_found",
			versions: []Version{
				{Version: "2.0.0", Properties: []Property{{Key: EngineProperty, Value: "^1.80.0"}}},
			},
			version:   domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
		{
			name:      "empty_versions",
			versions:  []Version{},
			version:   domain.Version{Major: 1, Minor: 0, Patch: 0},
			vscodeVer: domain.Version{Major: 1, Minor: 90, Patch: 0},
			platform:  domain.LinuxX64,
			wantFound: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, found := findSpecificSupportedVersion(testCase.versions, testCase.version, testCase.vscodeVer, testCase.platform)
			if found != testCase.wantFound {
				t.Fatalf("found = %v, want %v", found, testCase.wantFound)
			}
			if found && got.Version != testCase.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, testCase.wantVersion)
			}
			if found && got.TargetPlatform != testCase.wantTargetPlatform {
				t.Errorf("targetPlatform = %q, want %q", got.TargetPlatform, testCase.wantTargetPlatform)
			}
		})
	}
}

func TestGetDownloadInfoWithVersion(t *testing.T) {
	const testSize int64 = 12345

	tests := []struct {
		name        string
		response    string
		version     *domain.Version
		platform    domain.Platform
		wantVersion domain.Version
		wantErr     bool
	}{
		{
			name:     "specific_version_found",
			platform: domain.LinuxX64,
			version:  &domain.Version{Major: 1, Minor: 0, Patch: 0},
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0", "assetUri": "{{BASE_URL}}/ext/2.0.0"},
							{"version": "1.0.0", "assetUri": "{{BASE_URL}}/ext/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 1, Minor: 0, Patch: 0},
		},
		{
			name:     "specific_version_not_found",
			platform: domain.LinuxX64,
			version:  &domain.Version{Major: 9, Minor: 9, Patch: 9},
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0", "assetUri": "{{BASE_URL}}/ext/2.0.0"},
							{"version": "1.0.0", "assetUri": "{{BASE_URL}}/ext/1.0.0"}
						]
					}]
				}]
			}`,
			wantErr: true,
		},
		{
			name:     "nil_version_returns_latest",
			platform: domain.LinuxX64,
			version:  nil,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0", "assetUri": "{{BASE_URL}}/ext/2.0.0"},
							{"version": "1.0.0", "assetUri": "{{BASE_URL}}/ext/1.0.0"}
						]
					}]
				}]
			}`,
			wantVersion: domain.Version{Major: 2, Minor: 0, Patch: 0},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var serverURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.Header().Set("Content-Length", strconv.FormatInt(testSize, 10))
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusOK)
				response := strings.ReplaceAll(testCase.response, "{{BASE_URL}}", serverURL)
				w.Write([]byte(response))
			}))
			defer server.Close()
			serverURL = server.URL

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, testCase.platform, 5*time.Second, 15*time.Second, nil)
			_, got, err := registry.GetDownloadInfo(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"}, testCase.version)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if testCase.wantErr {
				return
			}

			if got.Version != testCase.wantVersion {
				t.Errorf("Version: got %+v, want %+v", got.Version, testCase.wantVersion)
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

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
			versionInfo := domain.DownloadInfo{
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

	registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
	versionInfo := domain.DownloadInfo{
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

		registry := NewRegistry("", failServer.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
		versionInfo := domain.DownloadInfo{
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

		registry := NewRegistry("", failServer.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
		versionInfo := domain.DownloadInfo{
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
		registry := NewRegistry("", stallServer.Client(), vscodeVer, domain.LinuxX64, 100*time.Millisecond, 15*time.Second, nil)
		versionInfo := domain.DownloadInfo{
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

		registry := NewRegistry("", failServer.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 15*time.Second, nil)
		versionInfo := domain.DownloadInfo{
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

func TestGetSize(t *testing.T) {
	t.Run("ok_source", func(t *testing.T) {
		size := 10
		body := "abcdefghij" // 10 байт
		okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(size))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(body))
		}))
		defer okServer.Close()

		registry := NewRegistry("", okServer.Client(), vscodeVer, domain.LinuxX64, 100*time.Millisecond, 15*time.Second, nil)
		ctx := t.Context()
		got, err := registry.getSize(ctx, []string{okServer.URL})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != int64(size) {
			t.Errorf("got %d, want %d", got, size)
		}

	})

	t.Run("source_fails_fallback_succeeds", func(t *testing.T) {
		size := 10
		body := "abcdefghij" // 10 байт
		okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(size))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(body))
		}))
		defer okServer.Close()

		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failServer.Close()

		stallServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer stallServer.Close()

		invalidURL := "://example.com"

		client := &http.Client{
			Timeout: 10 * time.Millisecond,
		}
		registry := NewRegistry("", client, vscodeVer, domain.LinuxX64, 100*time.Millisecond, 15*time.Second, nil)
		ctx := t.Context()
		got, err := registry.getSize(ctx, []string{stallServer.URL, failServer.URL, invalidURL, okServer.URL})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != int64(size) {
			t.Errorf("got %d, want %d", got, size)
		}
	})

	t.Run("all_sources_fail", func(t *testing.T) {
		failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer failServer.Close()

		client := &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 10 * time.Millisecond,
			},
		}
		registry := NewRegistry("", client, vscodeVer, domain.LinuxX64, 100*time.Millisecond, 15*time.Second, nil)
		ctx := t.Context()
		_, err := registry.getSize(ctx, []string{failServer.URL, failServer.URL})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrAllSourcesUnavailable) {
			t.Errorf("got error %v, expected error %v", err, domain.ErrAllSourcesUnavailable)
		}
	})
}

func TestExtensionQueryRetry(t *testing.T) {
	validResponse := `{"results": [{"extensions": [], "resultMetadata": []}]}`

	t.Run("retries_on_server_error", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)
			if count < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(validResponse))
		}))
		defer server.Close()

		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 5*time.Second, nil)
		_, err := registry.Search(context.Background(), domain.SearchQuery{Query: "test", Limit: 10, Type: domain.SearchByText})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := requestCount.Load(); got != 3 {
			t.Errorf("request count: got %d, want 3", got)
		}
	})

	t.Run("no_retry_on_client_error", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 5*time.Second, nil)
		_, err := registry.Search(context.Background(), domain.SearchQuery{Query: "test", Limit: 10, Type: domain.SearchByText})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := requestCount.Load(); got != 1 {
			t.Errorf("request count: got %d, want 1", got)
		}
	})

	t.Run("retries_on_timeout", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)
			if count < 3 {
				// Ждём дольше чем queryTimeout, чтобы клиент получил таймаут
				select {
				case <-time.After(500 * time.Millisecond):
				case <-r.Context().Done():
				}
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(validResponse))
		}))
		defer server.Close()

		// Короткий queryTimeout для быстрого теста
		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 50*time.Millisecond, nil)
		_, err := registry.Search(context.Background(), domain.SearchQuery{Query: "test", Limit: 10, Type: domain.SearchByText})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := requestCount.Load(); got != 3 {
			t.Errorf("request count: got %d, want 3", got)
		}
	})

	t.Run("exhausts_all_retries", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 5*time.Second, nil)
		_, err := registry.Search(context.Background(), domain.SearchQuery{Query: "test", Limit: 10, Type: domain.SearchByText})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := requestCount.Load(); got != int32(DefaultQueryRetries) {
			t.Errorf("request count: got %d, want %d", got, DefaultQueryRetries)
		}
	})

	t.Run("parent_context_cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		registry := NewRegistry(server.URL, server.Client(), vscodeVer, domain.LinuxX64, 5*time.Second, 5*time.Second, nil)
		_, err := registry.Search(ctx, domain.SearchQuery{Query: "go", Limit: 10, Type: domain.SearchByText})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetVersions(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		platform   domain.Platform
		limit      int
		want       []domain.VersionInfo
		wantErr    bool
	}{
		{
			name:       "multiple_versions",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "go",
						"publisher": {"publisherName": "golang"},
						"versions": [
							{"version": "3.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.Engine", "value": "^1.80.0"}]},
							{"version": "2.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.Engine", "value": "^1.80.0"}]},
							{"version": "1.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.Engine", "value": "^1.80.0"}]}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 3, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "deduplicates_platform_variants",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "1.0.0", "targetPlatform": "linux-x64"},
							{"version": "1.0.0", "targetPlatform": "darwin-arm64"},
							{"version": "1.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "engine_incompatible",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.Engine", "value": "^2.0.0"}]},
							{"version": "1.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.Engine", "value": "^1.80.0"}]}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: false, PlatformCompatible: true},
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "platform_incompatible",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0", "targetPlatform": "darwin-arm64"},
							{"version": "1.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: false},
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "skips_all_prerelease_versions",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "3.0.0", "properties": [{"key": "Microsoft.VisualStudio.Code.PreRelease", "value": "true"}]},
							{"version": "2.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "limit_applied",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			limit:      2,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "3.0.0"},
							{"version": "2.0.0"},
							{"version": "1.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 3, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "limit_zero_returns_all",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			limit:      0,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "2.0.0"},
							{"version": "1.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 2, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "platform_compatible_via_universal_fallback",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response: `{
				"results": [{
					"extensions": [{
						"extensionName": "ext",
						"publisher": {"publisherName": "test"},
						"versions": [
							{"version": "1.0.0", "targetPlatform": "darwin-arm64"},
							{"version": "1.0.0"}
						]
					}]
				}]
			}`,
			want: []domain.VersionInfo{
				{Version: domain.Version{Major: 1, Minor: 0, Patch: 0}, VscodeCompatible: true, PlatformCompatible: true},
			},
		},
		{
			name:       "empty_versions",
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
			want: nil,
		},
		{
			name:       "extension_not_found",
			statusCode: http.StatusOK,
			platform:   domain.LinuxX64,
			response:   `{"results": [{"extensions": []}]}`,
			wantErr:    true,
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

			registry := NewRegistry(server.URL, server.Client(), vscodeVer, testCase.platform, 5*time.Second, 15*time.Second, nil)
			got, err := registry.GetVersions(context.Background(), domain.ExtensionID{Publisher: "test", Name: "ext"}, testCase.limit)

			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !testCase.wantErr && !reflect.DeepEqual(got, testCase.want) {
				t.Errorf("got %+v, want %+v", got, testCase.want)
			}
		})
	}
}
