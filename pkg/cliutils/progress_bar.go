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

// Создаёт стиль прогресс бара по наименованию
func NewProgressBarStyle(name string, terminalWidth int) (ProgressBarStyle, error) {
	switch name {
	case "pacman":
		return NewPacmanProgressBar(terminalWidth), nil
	default:
		return nil, fmt.Errorf("new progress bar style: unknown style %q", name)
	}
}

// Стиль как у pacman
type PacmanProgressStyle struct {
	terminalWidth int
}

func NewPacmanProgressBar(terminalWidth int) PacmanProgressStyle {
	return PacmanProgressStyle{terminalWidth}
}

const (
	labelWidth    = 35
	fixedOverhead = labelWidth + 3 + 3 + 4 + 2 + 23 // label(35) + "  ["(3) + "]  "(3) + "100%"(4) + "  "(2) + запас на счётчики размера(23)
)

// Обрезает или дополняет пробелами label до фиксированной ширины
func padOrTruncate(s string, width int) string {
	if len(s) > width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (pb PacmanProgressStyle) Draw(label string, downloaded, total int64) string {
	fixedLabel := padOrTruncate(label, labelWidth)
	downloadedMB := float64(downloaded) / 1024 / 1024
	totalMB := float64(total) / 1024 / 1024

	// Если не знаем итоговый вес
	if total <= 0 {
		return fmt.Sprintf("%s  %.1f MiB", fixedLabel, downloadedMB)
	}

	downloadedPercent := int(downloaded * 100 / total)
	barWidth := pb.terminalWidth - fixedOverhead
	if barWidth < 1 {
		return fmt.Sprintf("%s  %d%%  %.1f MiB / %.1f MiB", fixedLabel, downloadedPercent, downloadedMB, totalMB)
	}

	filledBucket := downloadedPercent * barWidth / 100
	progressLine := strings.Repeat("#", filledBucket) + strings.Repeat(" ", barWidth-filledBucket)
	return fmt.Sprintf("%s  [%s]  %d%%  %.1f MiB / %.1f MiB", fixedLabel, progressLine, downloadedPercent, downloadedMB, totalMB)
}
