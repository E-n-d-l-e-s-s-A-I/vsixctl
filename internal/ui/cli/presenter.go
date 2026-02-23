package cli

import (
	"fmt"
	"io"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/ui"
)

type CliPresenter struct {
	out io.Writer
}

func NewPresenter(out io.Writer) *CliPresenter {
	return &CliPresenter{out}
}

func (presenter *CliPresenter) ShowExtensions(extensions []domain.Extension) {
	for i, extension := range extensions {
		fmt.Fprintf(presenter.out, "%d. %s - %s\n", i+1, extension.Name, extension.Description)
	}
	if len(extensions) == 0 {
		fmt.Fprintf(presenter.out, "no results\n")
	}
}

func (presenter *CliPresenter) ShowSearchResults(results []domain.SearchResult) {

}

func (presenter *CliPresenter) StartProgress(label string) (domain.ProgressFunc, ui.FinishFunc) {
	return nil, nil
}

func (presenter *CliPresenter) ShowMessage(msg string) {

}

func (presenter *CliPresenter) ShowError(err error) {

}
