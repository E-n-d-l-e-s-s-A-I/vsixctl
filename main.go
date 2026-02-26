package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/cmd"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/config"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/detect"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/registry/marketplace"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/storage/vscode"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui/cli"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
)

func main() {
	// Парсинг конфига
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	cfgPath := config.DefaultPath(homeDir, os.Getenv("XDG_CONFIG_HOME"))
	platform := detect.DetectPlatform(runtime.GOOS, runtime.GOARCH)
	vscodeExtensionsEnv := os.Getenv("VSCODE_EXTENSIONS")
	extensionsDir := detect.DetectExtensionsDir(homeDir, vscodeExtensionsEnv)
	cfg, err := config.LoadOrCreate(cfgPath, platform, extensionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	// TODO убрать хардкод url маркетплейса
	registry := marketplace.NewRegistry("https://marketplace.visualstudio.com/_apis/public/gallery", client)
	storage := vscode.NewVSCodeStorage(cfg.ExtensionsPath)
	userCase := usecases.NewUserCaseService(registry, storage)
	app := &cmd.App{
		UseCase:   userCase,
		Presenter: cli.NewPresenter(os.Stdout),
	}

	if err := cmd.NewRootCmd(app).Execute(); err != nil {
		os.Exit(1)
	}
}
