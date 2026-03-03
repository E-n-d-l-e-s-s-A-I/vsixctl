package cli

import (
	"fmt"
	"strings"
)

type ProgressBar interface {
	Draw(label string, downloaded, total int64) string
}

type PacmanProgressBar struct {
	width int
}

func NewPacmanProgressBar(width int) PacmanProgressBar {
	return PacmanProgressBar{width}
}

func (pb PacmanProgressBar) Draw(label string, downloaded, total int64) string {
	downloadedMB := float64(downloaded) / 1024 / 1024
	totalMB := float64(total) / 1024 / 1024

	// Если не знаем итоговый вес
	if total <= 0 {
		return fmt.Sprintf("%s  %.1f MiB", label, downloadedMB)

	}

	downloadedPercent := int(downloaded * 100 / total)
	filledBucket := downloadedPercent * pb.width / 100
	downloadedLine := strings.Repeat("#", filledBucket)
	restLine := strings.Repeat(" ", pb.width-filledBucket)

	progressLine := downloadedLine + restLine
	return fmt.Sprintf("%s  [%s]  %d%%  %.1f MiB / %.1f MiB", label, progressLine, downloadedPercent, downloadedMB, totalMB)
}
