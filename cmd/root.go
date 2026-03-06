package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/config"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/detect"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/registry/marketplace"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/storage/vscode"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui/cli"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type App struct {
	UseCase   usecases.UseCase
	Presenter ui.Presenter
}

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	var app App
	var verbose bool

	root := &cobra.Command{
		Use:   "vsixctl",
		Short: "Fast extension manager for VS Code",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("get home dir: %w", err)
			}
			cfgPath := config.DefaultPath(homeDir, os.Getenv("XDG_CONFIG_HOME"))
			platform := detect.DetectPlatform(runtime.GOOS, runtime.GOARCH)
			extensionsDir := detect.DetectExtensionsDir(homeDir, os.Getenv("VSCODE_EXTENSIONS"))
			cfg, err := config.LoadOrCreate(cfgPath, platform, extensionsDir)
			if err != nil {
				return err
			}

			progressBarStyle, err := cliutils.NewProgressBarStyle(cfg.ProgressBarStyle)
			if err != nil {
				return err
			}
			// Эта функция вызывается на каждой итерации отрисовки
			// term.GetSize() делает системный вызов для получения ширины терминала
			// Чтобы уменьшить кол-во системных вызовов можно обрабатывать сигнал SIGWINCH
			termWidth := func() int {
				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil {
					width = 80
				}
				return width
			}
			presenter := cli.NewPresenter(os.Stdout, termWidth, cli.DefaultRedrawInterval, progressBarStyle, verbose)

			registry := marketplace.NewRegistry(
				marketplace.DefaultURL,
				marketplace.NewDefaultHTTPClient(),
				platform,
				time.Duration(cfg.SourceTimeout),
				presenter.Log,
			)
			storage := vscode.NewVSCodeStorage(cfg.ExtensionsPath, presenter.Log)

			app.UseCase = usecases.NewUserCaseService(registry, storage, cfg.Parallelism)
			app.Presenter = presenter

			return nil
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	root.AddCommand(newVersionCommand())
	root.AddCommand(newSearchCommand(&app))
	root.AddCommand(newListCommand(&app))
	root.AddCommand(newInstallCommand(&app))
	return root
}
