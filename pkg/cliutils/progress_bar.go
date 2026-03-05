package cliutils

import (
	"fmt"
	"strings"
	"sync"
)

// Прогресс бар
// Реализует интерфейс Widget
type ProgressBar struct {
	mu         sync.Mutex
	label      string
	downloaded int64
	total      int64
	finish     bool
	style      ProgressBarStyle
}

func (w *ProgressBar) Render() (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.style.Draw(w.label, w.downloaded, w.total), !w.finish
}

func (w *ProgressBar) OnProgress(downloaded, total int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.downloaded = downloaded
	w.total = total
}

func (w *ProgressBar) OnFinish() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.finish = true
}

func NewProgressBar(label string, style ProgressBarStyle) *ProgressBar {
	return &ProgressBar{label: label, style: style}
}

// Стиль прогресс бара
type ProgressBarStyle interface {
	Draw(label string, downloaded, total int64) string
}

// Стиль как у pacman
type PacmanProgressStyle struct {
	width int
}

func NewPacmanProgressBar(width int) PacmanProgressStyle {
	return PacmanProgressStyle{width}
}

func (pb PacmanProgressStyle) Draw(label string, downloaded, total int64) string {
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
