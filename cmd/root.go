package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui"
	"github.com/spf13/cobra"
)

type App struct {
	Registry  domain.Registry
	Presenter ui.Presenter
}

func NewRootCmd(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "vsixctl",
		Short: "Fast extension manager for VS Code",
	}
	root.AddCommand(newVersionCommand())
	root.AddCommand(NewSearchCommand(app))
	return root
}
