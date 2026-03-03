package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Менеджер прогресс баров, отвечающий за асинхронную отрисовку в io.Writer
type ProgressManager struct {
	out            io.Writer     // Поток вывода
	redrawInterval time.Duration // Интервал отрисовки
	bars           []*barState   // Слайс отслеживаемых прогресс баров
	progressBar    ProgressBar   // Стиль отрисовки прогресс бара

	activeBars    int          // Кол-во активных прогресс баров
	lastLineCount int          // Кол-во строк отрисованных на прошлой итерации
	mu            sync.Mutex   // Мьютекс для синхронизации доступа к общему состоянию
	ticker        *time.Ticker // Тикер для цикла отрисовки

	stopChan chan struct{} // Канал для остановки цикла отрисовки
	done     chan struct{} // Канал для ответа что цикл завершился
}

// Состояние прогресс бара
type barState struct {
	label      string
	downloaded int64
	total      int64
	finish     bool
}

func NewProgressManager(out io.Writer, interval time.Duration, progressBar ProgressBar) *ProgressManager {
	return &ProgressManager{
		out:            out,
		redrawInterval: interval,
		progressBar:    progressBar,
	}
}

// Добавляет прогресс бар в менеджер
func (pm *ProgressManager) AddBar(label string) (domain.ProgressFunc, func()) {
	bar := barState{
		label:      label,
		downloaded: 0,
		total:      0,
	}

	// Критическая секция с изменением состояния ProgressManager
	// Добавляем новый progress bar в слайс отслеживаемых
	// И увеличиваем кол-во активных прогресс баров
	// Если кол-во активных прогресс баров увеличилось с 0 до 1, запускаем цикл отрисовки
	pm.mu.Lock()
	pm.bars = append(pm.bars, &bar)
	pm.activeBars += 1
	shouldStart := pm.activeBars == 1
	if shouldStart {
		pm.startTicker()
	}
	pm.mu.Unlock()

	// Колбек вызываемый в процессе загрузки контента, к которому привязан прогресс бар
	progressFunc := func(downloaded, total int64) {
		// Обновляем состояние прогресс бара
		pm.mu.Lock()
		bar.downloaded = downloaded
		bar.total = total
		pm.mu.Unlock()
	}

	// Колбек вызываемый при завершении загрузки контента, к которому привязан прогресс бар
	finish := func() {
		// Прекращаем отслеживание прогресс бара
		// И перерисовываем прогресс бары, чтобы отобразить полностью заполненный прогресс бар
		pm.mu.Lock()
		bar.finish = true
		pm.activeBars -= 1
		pm.redrawLocked()
		shouldStop := pm.activeBars == 0
		// Забираем локальные копии каналов и тикера до освобождения мьютекса,
		// чтобы stopTicker не конкурировал с startTicker за доступ к полям
		stopChan := pm.stopChan
		done := pm.done
		ticker := pm.ticker
		pm.mu.Unlock()

		// Если прогресс бары для отслеживания закончились, останавливаем цикл отрисовки
		if shouldStop {

			// Отправляем сигнал о завершении цикла
			close(stopChan)
			// Дожидаемся завершения цикла
			<-done
			ticker.Stop()
		}
	}

	return progressFunc, finish
}

// Запускает цикл отрисовки с интервалом redrawInterval
// И каналом остановки stopChan
func (pm *ProgressManager) startTicker() {
	ticker := time.NewTicker(pm.redrawInterval)
	stopChan := make(chan struct{})
	done := make(chan struct{})

	pm.ticker = ticker
	pm.stopChan = stopChan
	pm.done = done

	go func() {
		defer close(done)
		for {
			select {
			case <-ticker.C:
				pm.redraw()

			case <-stopChan:
				return
			}
		}
	}()
}

// Основная функция отрисовки
func (pm *ProgressManager) redraw() {
	pm.mu.Lock()
	pm.redrawLocked()
	pm.mu.Unlock()
}

// Функция отрисовки без блокировки мьютекса
// Для ситуаций когда вызывающий код уже захватил блокировку
func (pm *ProgressManager) redrawLocked() {
	if pm.lastLineCount > 0 {
		fmt.Fprintf(pm.out, "\033[%dA", pm.lastLineCount)
	}

	for _, bar := range pm.bars {
		fmt.Fprintf(pm.out, "%s\033[K\n", pm.progressBar.Draw(bar.label, bar.downloaded, bar.total))
	}
	pm.lastLineCount = len(pm.bars)
}
