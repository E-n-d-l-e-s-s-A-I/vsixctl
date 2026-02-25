package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/platform"
)

type Config struct {
	ExtensionsPath string            `json:"extensionsPath"`
	Platform       platform.Platform `json:"platform"`
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

func DefaultPath(homeDir string) string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		return filepath.Join(homeDir, ".config", "vsixctl", "config.json")
	}
	return filepath.Join(configDir, "vsixctl", "config.json")
}

func LoadOrCreate(path string, plt platform.Platform, homeDir string) (Config, error) {
	_, err := os.Stat(path)
	if err == nil {
		return Load(path)

	} else {
		// TODO убрать хардкод и добавить детектор
		cfg := Config{
			ExtensionsPath: filepath.Join(homeDir, ".vscode", "extensions"),
			Platform:       plt,
		}
		err = Save(path, cfg)
		if err != nil {
			return Config{}, err
		}
		return cfg, nil
	}
}
