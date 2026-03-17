package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

const (
	DefaultParallelism   = 3
	DefaultSourceTimeout = Duration(2 * time.Second)
	DefaultQueryTimeout  = Duration(7 * time.Second)
	DefaultProgressStyle = "pacman"
)

var validProgressBarStyles = []string{"pacman"}

type Config struct {
	ExtensionsPath   string          `json:"extensionsPath"`
	Platform         domain.Platform `json:"platform"`
	Parallelism      int             `json:"parallelism"`
	SourceTimeout    Duration        `json:"sourceTimeout"`
	QueryTimeout     Duration        `json:"queryTimeout"`
	ProgressBarStyle string          `json:"progressBarStyle"`
}

// Заполняет нулевые значения дефолтами для обратной совместимости со старыми конфигами
func (c *Config) applyDefaults() {
	if c.Parallelism == 0 {
		c.Parallelism = DefaultParallelism
	}
	if c.SourceTimeout == 0 {
		c.SourceTimeout = DefaultSourceTimeout
	}
	if c.QueryTimeout == 0 {
		c.QueryTimeout = DefaultQueryTimeout
	}
	if c.ProgressBarStyle == "" {
		c.ProgressBarStyle = DefaultProgressStyle
	}
}

func (c Config) Validate() error {
	if c.Parallelism <= 0 {
		return fmt.Errorf("validate config: parallelism must be >0")
	}

	if c.SourceTimeout <= 0 {
		return fmt.Errorf("validate config: sourceTimeout must be >0")
	}

	if c.QueryTimeout <= 0 {
		return fmt.Errorf("validate config: queryTimeout must be >0")
	}

	if !slices.Contains(validProgressBarStyles, c.ProgressBarStyle) {
		return fmt.Errorf("validate config: progressBarStyle must be one of: %s", strings.Join(validProgressBarStyles, ", "))
	}

	if !domain.IsValidPlatform(c.Platform) {
		return fmt.Errorf("validate config: invalid platform %q", c.Platform)
	}

	if c.ExtensionsPath == "" {
		return fmt.Errorf("validate config: extensionsPath must be set")
	}
	info, err := os.Stat(c.ExtensionsPath)
	if err != nil {
		return fmt.Errorf("validate config: extensionsPath: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("validate config: extensionsPath %q: is not a directory", c.ExtensionsPath)
	}
	return nil
}

func Load(path string) (Config, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	var cfg Config
	err = json.Unmarshal(fileContent, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.applyDefaults()
	err = cfg.Validate()
	if err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func DefaultPath(homeDir string, xdgConfigHome string) string {
	configDir := xdgConfigHome
	if configDir == "" {
		return filepath.Join(homeDir, ".config", "vsixctl", "config.json")
	}
	return filepath.Join(configDir, "vsixctl", "config.json")
}

// LoadOrCreate загружает конфиг из файла. Если файл не существует - создаёт его с переданными значениями.
func LoadOrCreate(path string, plt domain.Platform, extensionsDir string) (Config, error) {
	_, err := os.Stat(path)
	if err == nil {
		return Load(path)
	}
	cfg := Config{
		ExtensionsPath:   extensionsDir,
		Platform:         plt,
		Parallelism:      DefaultParallelism,
		SourceTimeout:    DefaultSourceTimeout,
		QueryTimeout:     DefaultQueryTimeout,
		ProgressBarStyle: DefaultProgressStyle,
	}
	err = Save(path, cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
