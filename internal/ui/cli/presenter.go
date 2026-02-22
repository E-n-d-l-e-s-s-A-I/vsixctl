package cli

import (
	"fmt"
	"io"
	"strings"

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
	lines := make([]string, len(extensions))

	for i, extension := range extensions {
		line := fmt.Sprintf("%d. %s - %s", i+1, extension.Name, extension.Description)
		lines[i] = line
	}

	output := []byte(strings.Join(lines, "\n"))
	presenter.out.Write(output)
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
