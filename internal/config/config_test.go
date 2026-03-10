package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Хелпер для создания валидного конфига
func validConfig() Config {
	return Config{
		ExtensionsPath:   "/home/user/.vscode/extensions",
		Platform:         domain.LinuxX64,
		Parallelism:      3,
		SourceTimeout:    Duration(2 * time.Second),
		ProgressBarStyle: "pacman",
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
			modify:  func(c *Config) { c.Parallelism = 0 },
			wantErr: true,
		},
		{
			name:    "negative_parallelism",
			modify:  func(c *Config) { c.Parallelism = -1 },
			wantErr: true,
		},
		{
			name:    "zero_source_timeout",
			modify:  func(c *Config) { c.SourceTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "negative_source_timeout",
			modify:  func(c *Config) { c.SourceTimeout = Duration(-1 * time.Second) },
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
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfig()
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
		name            string
		parallelism     int
		sourceTimeout   Duration
		progressStyle   string
		wantParallelism int
		wantTimeout     Duration
		wantStyle       string
	}{
		{
			name:            "all_zero_values",
			parallelism:     0,
			sourceTimeout:   0,
			progressStyle:   "",
			wantParallelism: DefaultParallelism,
			wantTimeout:     DefaultSourceTimeout,
			wantStyle:       DefaultProgressStyle,
		},
		{
			name:            "custom_values_preserved",
			parallelism:     5,
			sourceTimeout:   Duration(10 * time.Second),
			progressStyle:   "pacman",
			wantParallelism: 5,
			wantTimeout:     Duration(10 * time.Second),
			wantStyle:       "pacman",
		},
		{
			name:            "partial_zero_values",
			parallelism:     0,
			sourceTimeout:   Duration(5 * time.Second),
			progressStyle:   "",
			wantParallelism: DefaultParallelism,
			wantTimeout:     Duration(5 * time.Second),
			wantStyle:       DefaultProgressStyle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := Config{
				Parallelism:      testCase.parallelism,
				SourceTimeout:    testCase.sourceTimeout,
				ProgressBarStyle: testCase.progressStyle,
			}
			cfg.applyDefaults()
			if cfg.Parallelism != testCase.wantParallelism {
				t.Errorf("parallelism: got %d, want %d", cfg.Parallelism, testCase.wantParallelism)
			}
			if cfg.SourceTimeout != testCase.wantTimeout {
				t.Errorf("sourceTimeout: got %v, want %v", cfg.SourceTimeout, testCase.wantTimeout)
			}
			if cfg.ProgressBarStyle != testCase.wantStyle {
				t.Errorf("progressBarStyle: got %q, want %q", cfg.ProgressBarStyle, testCase.wantStyle)
			}
		})
	}
}

// Проверяет что старый конфиг без новых полей загружается с дефолтными значениями
func TestLoadOldConfig(t *testing.T) {
	oldConfig := `{
		"extensionsPath": "/home/user/.vscode/extensions",
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

	if cfg.Parallelism != DefaultParallelism {
		t.Errorf("parallelism: got %d, want %d", cfg.Parallelism, DefaultParallelism)
	}
	if cfg.SourceTimeout != DefaultSourceTimeout {
		t.Errorf("sourceTimeout: got %v, want %v", cfg.SourceTimeout, DefaultSourceTimeout)
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
			content: `{"extensionsPath": "/tmp", "platform": "freebsd-x64", "parallelism": 3, "sourceTimeout": "2s", "progressBarStyle": "pacman"}`,
		},
		{
			name:    "negative_parallelism",
			content: `{"extensionsPath": "/tmp", "platform": "linux-x64", "parallelism": -1, "sourceTimeout": "2s", "progressBarStyle": "pacman"}`,
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
