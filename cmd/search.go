package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newSearchCommand(app *App) *cobra.Command {
	var (
		limit      int
		searchType string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Args:  cobra.ExactArgs(1),
		Short: "Search extension in marketplace",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			t, err := domain.ParseSearchType(searchType)
			if err != nil {
				return err
			}
			query := args[0]

			// Если поиск по id, сразу же делаем валидацию
			if t == domain.SearchByID {
				_, err = domain.ParseExtensionID(query)
				if err != nil {
					return err
				}
			}

			results, err := app.UseCase.Search(cmd.Context(), domain.SearchQuery{Query: query, Limit: limit, Type: t})
			if err != nil {
				return err
			}
			app.Presenter.ShowSearchResults(results)
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "results limit")
	cmd.Flags().StringVar(&searchType, "type", "text", "search type: text, id, name")
	return cmd
}
