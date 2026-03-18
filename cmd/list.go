package cmd

import (
	"github.com/spf13/cobra"
)

func newListCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "Show installed extensions",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()
			results, err := app.UseCase.List(cmd.Context())
			if err != nil {
				return err
			}
			app.Presenter.ShowExtensions(results)
			return nil
		},
	}
	return cmd
}
