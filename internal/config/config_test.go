package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Хелпер для создания валидного конфига
func validConfig(extensionsPath string) Config {
	return Config{
		ExtensionsPath:    extensionsPath,
		Platform:          domain.LinuxX64,
		Parallelism:       intPtr(3),
		SourceIdleTimeout: Duration(2 * time.Second),
		QueryTimeout:      Duration(7 * time.Second),
		QueryRetries:      intPtr(1),
		ProgressBarStyle:  "pacman",
	}
}

// Проверяет валидацию конфига: валидный конфиг и каждое невалидное поле по отдельности
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid_config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "zero_parallelism",
			modify:  func(c *Config) { c.Parallelism = intPtr(0) },
			wantErr: true,
		},
		{
			name:    "negative_parallelism",
			modify:  func(c *Config) { c.Parallelism = intPtr(-1) },
			wantErr: true,
		},
		{
			name:    "nil_parallelism",
			modify:  func(c *Config) { c.Parallelism = nil },
			wantErr: true,
		},
		{
			name:    "zero_source_idle_timeout",
			modify:  func(c *Config) { c.SourceIdleTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "negative_source_idle_timeout",
			modify:  func(c *Config) { c.SourceIdleTimeout = Duration(-1 * time.Second) },
			wantErr: true,
		},
		{
			name:    "zero_query_timeout",
			modify:  func(c *Config) { c.QueryTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "negative_query_timeout",
			modify:  func(c *Config) { c.QueryTimeout = Duration(-1 * time.Second) },
			wantErr: true,
		},
		{
			name:    "zero_query_retries",
			modify:  func(c *Config) { c.QueryRetries = intPtr(0) },
			wantErr: false,
		},
		{
			name:    "negative_query_retries",
			modify:  func(c *Config) { c.QueryRetries = intPtr(-1) },
			wantErr: true,
		},
		{
			name:    "nil_query_retries",
			modify:  func(c *Config) { c.QueryRetries = nil },
			wantErr: true,
		},
		{
			name:    "invalid_progress_bar_style",
			modify:  func(c *Config) { c.ProgressBarStyle = "unknown" },
			wantErr: true,
		},
		{
			name:    "empty_progress_bar_style",
			modify:  func(c *Config) { c.ProgressBarStyle = "" },
			wantErr: true,
		},
		{
			name:    "invalid_platform",
			modify:  func(c *Config) { c.Platform = "freebsd-x64" },
			wantErr: true,
		},
		{
			name:    "empty_extensions_path",
			modify:  func(c *Config) { c.ExtensionsPath = "" },
			wantErr: true,
		},
		{
			name:    "nonexistent_extensions_path",
			modify:  func(c *Config) { c.ExtensionsPath = "/nonexistent/path" },
			wantErr: true,
		},
		{
			name: "extensions_path_is_file",
			modify: func(c *Config) {
				filePath := filepath.Join(c.ExtensionsPath, "not-a-dir")
				os.WriteFile(filePath, nil, 0o644)
				c.ExtensionsPath = filePath
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfig(t.TempDir())
			testCase.modify(&cfg)
			err := cfg.Validate()
			if testCase.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Проверяет что нулевые значения заполняются дефолтами, а заданные пользователем сохраняются
func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name                  string
		parallelism           *int
		sourceIdleTimeout     Duration
		queryTimeout          Duration
		queryRetries          *int
		progressStyle         string
		wantParallelism       int
		wantSourceIdleTimeout Duration
		wantQueryTimeout      Duration
		wantQueryRetries      int
		wantStyle             string
	}{
		{
			name:                  "all_nil_values",
			parallelism:           nil,
			sourceIdleTimeout:     0,
			queryTimeout:          0,
			queryRetries:          nil,
			progressStyle:         "",
			wantParallelism:       DefaultParallelism,
			wantSourceIdleTimeout: DefaultSourceIdleTimeout,
			wantQueryTimeout:      DefaultQueryTimeout,
			wantQueryRetries:      DefaultQueryRetries,
			wantStyle:             DefaultProgressStyle,
		},
		{
			name:                  "custom_values_preserved",
			parallelism:           intPtr(5),
			sourceIdleTimeout:     Duration(10 * time.Second),
			queryTimeout:          Duration(20 * time.Second),
			queryRetries:          intPtr(5),
			progressStyle:         "pacman",
			wantParallelism:       5,
			wantSourceIdleTimeout: Duration(10 * time.Second),
			wantQueryTimeout:      Duration(20 * time.Second),
			wantQueryRetries:      5,
			wantStyle:             "pacman",
		},
		{
			name:                  "explicit_zero_preserved",
			parallelism:           intPtr(0),
			sourceIdleTimeout:     Duration(5 * time.Second),
			queryTimeout:          Duration(5 * time.Second),
			queryRetries:          intPtr(0),
			progressStyle:         "pacman",
			wantParallelism:       0,
			wantSourceIdleTimeout: Duration(5 * time.Second),
			wantQueryTimeout:      Duration(5 * time.Second),
			wantQueryRetries:      0,
			wantStyle:             "pacman",
		},
		{
			name:                  "partial_nil_values",
			parallelism:           nil,
			sourceIdleTimeout:     Duration(5 * time.Second),
			queryTimeout:          0,
			queryRetries:          nil,
			progressStyle:         "",
			wantParallelism:       DefaultParallelism,
			wantSourceIdleTimeout: Duration(5 * time.Second),
			wantQueryTimeout:      DefaultQueryTimeout,
			wantQueryRetries:      DefaultQueryRetries,
			wantStyle:             DefaultProgressStyle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := Config{
				Parallelism:       testCase.parallelism,
				SourceIdleTimeout: testCase.sourceIdleTimeout,
				QueryTimeout:      testCase.queryTimeout,
				QueryRetries:      testCase.queryRetries,
				ProgressBarStyle:  testCase.progressStyle,
			}
			cfg.applyDefaults()
			if *cfg.Parallelism != testCase.wantParallelism {
				t.Errorf("parallelism: got %d, want %d", *cfg.Parallelism, testCase.wantParallelism)
			}
			if cfg.SourceIdleTimeout != testCase.wantSourceIdleTimeout {
				t.Errorf("sourceIdleTimeout: got %v, want %v", cfg.SourceIdleTimeout, testCase.wantSourceIdleTimeout)
			}
			if cfg.QueryTimeout != testCase.wantQueryTimeout {
				t.Errorf("queryTimeout: got %v, want %v", cfg.QueryTimeout, testCase.wantQueryTimeout)
			}
			if *cfg.QueryRetries != testCase.wantQueryRetries {
				t.Errorf("queryRetries: got %d, want %d", *cfg.QueryRetries, testCase.wantQueryRetries)
			}
			if cfg.ProgressBarStyle != testCase.wantStyle {
				t.Errorf("progressBarStyle: got %q, want %q", cfg.ProgressBarStyle, testCase.wantStyle)
			}
		})
	}
}

// Проверяет что старый конфиг без новых полей загружается с дефолтными значениями
func TestLoadOldConfig(t *testing.T) {
	extDir := t.TempDir()
	oldConfig := `{
		"extensionsPath": "` + extDir + `",
		"platform": "linux-x64"
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(oldConfig), 0o644)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *cfg.Parallelism != DefaultParallelism {
		t.Errorf("parallelism: got %d, want %d", *cfg.Parallelism, DefaultParallelism)
	}
	if cfg.SourceIdleTimeout != DefaultSourceIdleTimeout {
		t.Errorf("sourceIdleTimeout: got %v, want %v", cfg.SourceIdleTimeout, DefaultSourceIdleTimeout)
	}
	if cfg.QueryTimeout != DefaultQueryTimeout {
		t.Errorf("queryTimeout: got %v, want %v", cfg.QueryTimeout, DefaultQueryTimeout)
	}
	if *cfg.QueryRetries != DefaultQueryRetries {
		t.Errorf("queryRetries: got %d, want %d", *cfg.QueryRetries, DefaultQueryRetries)
	}
	if cfg.ProgressBarStyle != DefaultProgressStyle {
		t.Errorf("progressBarStyle: got %q, want %q", cfg.ProgressBarStyle, DefaultProgressStyle)
	}
}

// Проверяет что Load возвращает ошибку при невалидном JSON и невалидных значениях полей
func TestLoadInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid_json",
			content: `{invalid`,
		},
		{
			name:    "invalid_platform",
			content: `{"extensionsPath": "/tmp", "platform": "freebsd-x64", "parallelism": 3, "sourceIdleTimeout": "2s", "progressBarStyle": "pacman"}`,
		},
		{
			name:    "negative_parallelism",
			content: `{"extensionsPath": "/tmp", "platform": "linux-x64", "parallelism": -1, "sourceIdleTimeout": "2s", "progressBarStyle": "pacman"}`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			err := os.WriteFile(path, []byte(testCase.content), 0o644)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			_, err = Load(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// Проверяет что Load возвращает ошибку при отсутствии файла
func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
