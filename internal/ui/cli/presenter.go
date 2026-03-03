package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type CliPresenter struct {
	out             io.Writer
	progressManager *ProgressManager
}

func NewPresenter(out io.Writer, redrawInterval time.Duration, progressBar ProgressBar) *CliPresenter {
	p := &CliPresenter{
		out: out,
	}
	p.progressManager = NewProgressManager(out, redrawInterval, progressBar)
	return p
}

func (p *CliPresenter) ShowExtensions(extensions []domain.Extension) {
	for i, extension := range extensions {
		fmt.Fprintf(p.out, "%d. %s - %s\n", i+1, extension.ID, extension.Description)
	}
	if len(extensions) == 0 {
		fmt.Fprintf(p.out, "no results\n")
	}
}

func (p *CliPresenter) ShowSearchResults(results []domain.SearchResult) {

}

// TODO написать тесты
func (p *CliPresenter) StartProgress(label string) (domain.ProgressFunc, func()) {
	return p.progressManager.AddBar(label)
}

func (p *CliPresenter) ShowMessage(msg string) {
	fmt.Fprint(p.out, msg+"\n")

}

func (p *CliPresenter) ShowError(err error) {
	fmt.Fprintf(p.out, "%s\n", err.Error())
}
