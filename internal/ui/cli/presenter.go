package cli

import (
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/cliutils"
)

type CliPresenter struct {
	in               io.Reader // Поток ввода
	terminalRenderer *cliutils.TerminalRenderer
	progressBarStyle cliutils.ProgressBarStyle
	debug            bool // Показывать ли логи
}

const DefaultRedrawInterval = 50 * time.Millisecond

func NewPresenter(out io.Writer, in io.Reader, outWidth func() int, redrawInterval time.Duration, progressBarStyle cliutils.ProgressBarStyle, debug bool) *CliPresenter {
	p := &CliPresenter{
		in:               in,
		progressBarStyle: progressBarStyle,
		debug:            debug,
	}
	p.terminalRenderer = cliutils.NewTerminalRenderer(out, outWidth, redrawInterval)
	return p
}

func (p *CliPresenter) ShowExtensions(extensions []domain.Extension) {
	for i, extension := range extensions {
		p.terminalRenderer.Log(formatExtension(i+1, extension))
	}
	if len(extensions) == 0 {
		p.terminalRenderer.Log("no results")
	}
}

func (p *CliPresenter) ShowInstallResult(res []domain.ExtensionResult) {
	p.showResult(res, "installed")
}

func (p *CliPresenter) ShowRemoveResult(res []domain.ExtensionResult) {
	p.showResult(res, "deleted")
}

func (p *CliPresenter) ShowUpdateResult(res []domain.ExtensionResult) {
	p.showResult(res, "updated")
}

func (p *CliPresenter) showResult(res []domain.ExtensionResult, successMsg string) {
	for _, r := range res {
		p.ShowMessage(formatResult(r, successMsg))
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
	if p.debug {
		p.ShowMessage(msg)
	}
}

func (p *CliPresenter) Wait() {
	p.terminalRenderer.Wait()
}

// ConfirmInstall подтверждает установку у пользователя
func (p *CliPresenter) ConfirmInstall(requestedIDs []domain.ExtensionID, extensions []domain.DownloadInfo, reinstall []domain.ReinstallInfo) bool {
	p.ShowMessage(formatInstallPlan(requestedIDs, extensions, reinstall))
	return p.confirm("Proceed with installation? [Y/n] ")
}

// ConfirmRemove подтверждает удаление у пользователя
func (p *CliPresenter) ConfirmRemove(requestedIds []domain.ExtensionID, extensions []domain.Extension) bool {
	p.ShowMessage(formatRemovePlan(requestedIds, extensions))
	return p.confirm("Proceed with removal? [Y/n] ")
}

// ConfirmUpdate подтверждает обновление у пользователя
func (p *CliPresenter) ConfirmUpdate(toUpdate []domain.UpdateInfo) bool {
	p.ShowMessage(formatUpdatePlan(toUpdate))
	return p.confirm("Proceed with update? [Y/n] ")
}

func (p *CliPresenter) confirm(prompt string) bool {
	p.ShowMessage(prompt)
	scanner := bufio.NewScanner(p.in)
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())
	return answer == "" || strings.EqualFold(answer, "y")
}
