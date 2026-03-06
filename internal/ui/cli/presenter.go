package cli

import (
	"io"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
)

type CliPresenter struct {
	terminalRenderer *cliutils.TerminalRenderer
	progressBarStyle cliutils.ProgressBarStyle
	verbose          bool // Показывать ли логи
}

const DefaultRedrawInterval = 50 * time.Millisecond

func NewPresenter(out io.Writer, outWidth func() int, redrawInterval time.Duration, progressBarStyle cliutils.ProgressBarStyle, verbose bool) *CliPresenter {
	p := &CliPresenter{
		progressBarStyle: progressBarStyle,
		verbose:          verbose,
	}
	p.terminalRenderer = cliutils.NewTerminalRenderer(out, outWidth, redrawInterval)
	return p
}

func (p *CliPresenter) ShowExtensions(extensions []domain.Extension) {
	for i, extension := range extensions {
		p.terminalRenderer.Log(FormatExtension(i+1, extension))
	}
	if len(extensions) == 0 {
		p.terminalRenderer.Log("no results")
	}
}

func (p *CliPresenter) ShowInstallResult(res []domain.InstallResult) {
	for _, r := range res {
		p.ShowMessage(FormatInstallResult(r))
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

func (p *CliPresenter) Log(msg string) {
	if p.verbose {
		p.ShowMessage(msg)
	}
}

func (p *CliPresenter) Wait() {
	p.terminalRenderer.Wait()
}
