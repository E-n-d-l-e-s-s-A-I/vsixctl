package cli

import (
	"fmt"
	"strings"
)

type ProgressBar interface {
	Draw(label string, downloaded, total int64) string
}

type PacmanProgressBar struct {
	bucketCount int
}

func NewPacmanProgressBar(bucketCount int) PacmanProgressBar {
	return PacmanProgressBar{bucketCount}
}

func (pb PacmanProgressBar) Draw(label string, downloaded, total int64) string {
	downloadedMB := float64(downloaded) / 1024 / 1024
	totalMB := float64(total) / 1024 / 1024

	// Если не знаем итоговый вес
	if total <= 0 {
		return fmt.Sprintf("%s  %.1f MiB", label, downloadedMB)

	}

	downloadedPercent := int(downloaded * 100 / total)
	filledBucket := downloadedPercent * pb.bucketCount / 100
	downloadedLine := strings.Repeat("#", filledBucket)
	restLine := strings.Repeat(" ", pb.bucketCount-filledBucket)

	progressLine := downloadedLine + restLine
	return fmt.Sprintf("%s  [%s]  %d%%  %.1f MiB / %.1f MiB", label, progressLine, downloadedPercent, downloadedMB, totalMB)
}
