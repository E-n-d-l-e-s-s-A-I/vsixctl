package main

import (
	"fmt"
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
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
	"golang.org/x/term"
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

	// Собираем зависимости
	registry := marketplace.NewRegistry(
		marketplace.DefaultURL,
		marketplace.NewDefaultHTTPClient(),
		platform,
		time.Duration(cfg.SourceTimeout),
	)
	storage := vscode.NewVSCodeStorage(cfg.ExtensionsPath)
	terminalWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		terminalWidth = 80
	}
	progressBarStyle, err := cliutils.NewProgressBarStyle(cfg.ProgressBarStyle, terminalWidth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	presenter := cli.NewPresenter(os.Stdout, cli.DefaultRedrawInterval, terminalWidth, progressBarStyle)

	userCase := usecases.NewUserCaseService(registry, storage, cfg.Parallelism)
	app := &cmd.App{
		UseCase:   userCase,
		Presenter: presenter,
	}

	if err := cmd.NewRootCmd(app).Execute(); err != nil {
		os.Exit(1)
	}
}
