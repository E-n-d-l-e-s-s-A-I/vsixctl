package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

type CliPresenter struct {
	out             io.Writer
	bucketCount     int
	progressManager *ProgressManager
}

func NewPresenter(out io.Writer, redrawInterval time.Duration, bucketCount int) *CliPresenter {
	p := &CliPresenter{
		out:         out,
		bucketCount: bucketCount,
	}
	p.progressManager = NewProgressManager(out, redrawInterval, p.PacmanStyleDraw)
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
// Pacman-style progress bar
func (p *CliPresenter) StartProgress(label string) (domain.ProgressFunc, func()) {
	return p.progressManager.AddBar(label)
}

func (p *CliPresenter) ShowMessage(msg string) {
	fmt.Fprint(p.out, msg+"\n")

}

func (p *CliPresenter) ShowError(err error) {
	fmt.Fprintf(p.out, "%s\n", err.Error())
}

// Pacman-style progress bar
// TODO вынести в отдельную сущность
// И Сделать di этой сущности
func (p *CliPresenter) PacmanStyleDraw(label string, downloaded, total int64) string {
	downloadedMB := float64(downloaded) / 1024 / 1024

	totalMB := float64(total) / 1024 / 1024
	// Если не знаем итоговый вес
	if total <= 0 {
		return fmt.Sprintf("%s  %.1f MiB", label, downloadedMB)

	}

	downloadedPercent := int(downloaded * 100 / total)
	filledBucket := downloadedPercent * p.bucketCount / 100
	downloadedLine := strings.Repeat("#", filledBucket)
	restLine := strings.Repeat(" ", p.bucketCount-filledBucket)

	progressLine := downloadedLine + restLine

	return fmt.Sprintf("%s  [%s]  %d%%  %.1f MiB / %.1f MiB", label, progressLine, downloadedPercent, downloadedMB, totalMB)
}
