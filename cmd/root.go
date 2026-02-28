package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

type App struct {
	UseCase   usecases.UseCase
	Presenter ui.Presenter
}

func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "vsixctl",
		Short: "Fast extension manager for VS Code",
	}
	root.AddCommand(newVersionCommand())
	root.AddCommand(newSearchCommand(app))
	root.AddCommand(newListCommand(app))
	root.AddCommand(newInstallCommand(app))
	return root
}
