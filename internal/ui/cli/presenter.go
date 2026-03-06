package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
)

type CliPresenter struct {
	terminalRenderer *cliutils.TerminalRenderer
	progressBarStyle cliutils.ProgressBarStyle
}

const DefaultRedrawInterval = 50 * time.Millisecond

func NewPresenter(out io.Writer, outWidth func() int, redrawInterval time.Duration, progressBarStyle cliutils.ProgressBarStyle) *CliPresenter {
	p := &CliPresenter{
		progressBarStyle: progressBarStyle,
	}
	p.terminalRenderer = cliutils.NewTerminalRenderer(out, outWidth, redrawInterval)
	return p
}

func (p *CliPresenter) ShowExtensions(extensions []domain.Extension) {
	for i, extension := range extensions {
		p.terminalRenderer.Log(fmt.Sprintf("%d. %s - %s", i+1, extension.ID, extension.Description))
	}
	if len(extensions) == 0 {
		p.terminalRenderer.Log("no results")
	}
}

func (p *CliPresenter) ShowSearchResults(results []domain.SearchResult) {

}

func (p *CliPresenter) StartProgress(label string) (domain.ProgressFunc, func()) {
	bar := cliutils.NewProgressBar(label, p.progressBarStyle)
	p.terminalRenderer.AddWidget(bar)
	return bar.OnProgress, bar.OnFinish
}

func (p *CliPresenter) ShowMessage(msg string) {
	p.terminalRenderer.Log(msg)
}

func (p *CliPresenter) ShowError(err error) {
	p.terminalRenderer.Log(err.Error())
}

func (p *CliPresenter) Wait() {
	p.terminalRenderer.Wait()
}
