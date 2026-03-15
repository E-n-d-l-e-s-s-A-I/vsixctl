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

// Version задаётся при сборке через ldflags:
// go build -ldflags "-X github.com/E-n-d-l-e-s-s-A-I/vsixctl/cmd.Version=1.2.3"
var Version = "dev"

type App struct {
	UseCase   usecases.UseCase
	Presenter ui.Presenter
}

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	var app App
	var debug bool

	root := &cobra.Command{
		Use:     "vsixctl",
		Short:   "Fast extension manager for VS Code",
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Устанавливаем SilenceUsage в true, чтобы usage выводился только при ошибках связанных с парсингом команды
			cmd.SilenceUsage = true
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("get home dir: %w", err)
			}
			cfgPath := config.DefaultPath(homeDir, os.Getenv("XDG_CONFIG_HOME"))
			platform := detect.DetectPlatform(runtime.GOOS, runtime.GOARCH)
			vscodeVer, err := detect.DetectVscodeVer(cmd.Context())
			if err != nil {
				return err
			}
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
			presenter := cli.NewPresenter(os.Stdout, os.Stdin, termWidth, cli.DefaultRedrawInterval, progressBarStyle, debug)

			registry := marketplace.NewRegistry(
				marketplace.DefaultURL,
				marketplace.NewDefaultHTTPClient(),
				vscodeVer,
				platform,
				time.Duration(cfg.SourceTimeout),
				presenter.Log,
			)
			storage := vscode.NewStorage(cfg.ExtensionsPath, presenter.Log)

			app.UseCase = usecases.NewUseCaseService(registry, storage, presenter.ShowMessage, cfg.Parallelism)
			app.Presenter = presenter

			return nil
		},
	}

	root.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	root.AddCommand(newSearchCommand(&app))
	root.AddCommand(newListCommand(&app))
	root.AddCommand(newInstallCommand(&app))
	root.AddCommand(newRemoveCommand(&app))
	root.AddCommand(newUpdateCommand(&app))
	return root
}
