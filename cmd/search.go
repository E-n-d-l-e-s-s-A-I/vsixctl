package cmd

import "github.com/spf13/cobra"

func newSearchCommand(app *App) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Args:  cobra.ExactArgs(1),
		Short: "Search extension in marketplace",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()
			results, err := app.UseCase.Search(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			app.Presenter.ShowSearchResults(results)
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "results limit")
	return cmd
}
