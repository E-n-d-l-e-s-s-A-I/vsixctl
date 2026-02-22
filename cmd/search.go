package cmd

import "github.com/spf13/cobra"

func newSearchCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Args:  cobra.ExactArgs(1),
		Short: "Search extension in marketplace",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := app.UseCase.Search(cmd.Context(), args[0], 10)
			if err != nil {
				return err
			}
			app.Presenter.ShowExtensions(results)
			return nil
		},
	}
	return cmd
}
