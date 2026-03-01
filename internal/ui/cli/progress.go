package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// TODO Добавить в этот модуль комментарий, тесты и особое внимание уделить race conditions

type DrawFunction func(label string, downloaded, total int64) string

type ProgressManager struct {
	out           io.Writer
	interval      time.Duration
	bars          []*progressBar
	mu            sync.Mutex
	ticker        *time.Ticker
	drawFunc      DrawFunction
	lastLineCount int
	activeBars    int
	stopChan      chan struct{}
}

type progressBar struct {
	label      string
	downloaded int64
	total      int64
	finish     bool
}

func NewProgressManager(out io.Writer, interval time.Duration, drawFunc DrawFunction) *ProgressManager {
	return &ProgressManager{
		out:      out,
		interval: interval,
		bars:     []*progressBar{},
		mu:       sync.Mutex{},
		drawFunc: drawFunc,
	}
}

func (pm *ProgressManager) AddBar(label string) (domain.ProgressFunc, func()) {
	bar := progressBar{
		label:      label,
		downloaded: 0,
		total:      0,
	}

	pm.mu.Lock()
	pm.bars = append(pm.bars, &bar)
	pm.activeBars += 1
	shouldStart := pm.activeBars == 1
	if shouldStart {
		pm.startTicker()
	}
	pm.mu.Unlock()

	progressFunc := func(downloaded, total int64) {
		pm.mu.Lock()
		bar.downloaded = downloaded
		bar.total = total
		pm.mu.Unlock()
	}

	finish := func() {
		pm.mu.Lock()
		bar.finish = true
		pm.activeBars -= 1
		// Финальная отрисовка
		pm.redrawLocked()
		if pm.activeBars == 0 {
			pm.stopTicker()
		}
		pm.mu.Unlock()
	}

	return progressFunc, finish
}

func (pm *ProgressManager) startTicker() {
	pm.ticker = time.NewTicker(pm.interval)
	pm.stopChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.redraw()

			case <-pm.stopChan:
				return
			}
		}
	}()
}

func (pm *ProgressManager) stopTicker() {
	close(pm.stopChan)
	pm.ticker.Stop()
}

func (pm *ProgressManager) redraw() {
	pm.mu.Lock()
	pm.redrawLocked()
	pm.mu.Unlock()
}

func (pm *ProgressManager) redrawLocked() {
	if pm.lastLineCount > 0 {
		fmt.Fprintf(pm.out, "\033[%dA", pm.lastLineCount)
	}

	for _, bar := range pm.bars {
		fmt.Fprintf(pm.out, "%s\033[K\n", pm.drawFunc(bar.label, bar.downloaded, bar.total))
	}
	pm.lastLineCount = len(pm.bars)
}
