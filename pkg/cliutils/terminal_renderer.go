package cliutils

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Интерфейс виджета, с которыми работает TerminalRenderer
type Widget interface {
	// Рендер виджета. string - содержимое виджета,
	// bool - нужны ли ещё отрисовки:
	//   true - виджет еще нужно рисовать
	//   false - виджет больше не нужно рисовать
	Render() (string, bool)
}

// Полностью отвечает за вывод в терминал, нужен чтобы синхронизировать запись в поток вывода
// Поддерживает как динамические виджеты, которые должны реализовывать интерфейс Widget
// Так и вывод обычных сообщений
type TerminalRenderer struct {
	out             io.Writer     // Поток вывода
	redrawInterval  time.Duration // Интервал отрисовки
	widgets         []Widget      // Слайс отслеживаемых виджетов
	pendingMessages []string      // Очередь сообщений(логов) для отрисовки
	loopRunning     bool          // Запущен ли цикл отрисовки

	lastLineCount int           // Кол-во строк отрисованных на прошлой итерации
	mu            sync.Mutex    // Мьютекс для синхронизации доступа к общему состоянию
	done          chan struct{} // Канал завершения
}

func NewTerminalRenderer(out io.Writer, interval time.Duration) *TerminalRenderer {
	return &TerminalRenderer{
		out:            out,
		redrawInterval: interval,
	}
}

// Добавляет виджет
func (r *TerminalRenderer) AddWidget(widget Widget) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.widgets = append(r.widgets, widget)
	shouldStart := !r.loopRunning
	if shouldStart {
		r.startLoop()
	}
}

// Запускает цикл отрисовки с интервалом redrawInterval
func (r *TerminalRenderer) startLoop() {
	ticker := time.NewTicker(r.redrawInterval)
	r.loopRunning = true
	r.done = make(chan struct{})

	// Цикл отрисовки
	go func() {
		// В конце цикла стопаем тикер и закрываем канал завершения
		defer ticker.Stop()
		defer close(r.done)

		for {
			<-ticker.C

			// Захватываем блокировку на время итерации цикла
			r.mu.Lock()

			hasActive := r.redrawLocked()

			// Если нет активных виджетов, останавливаем цикл
			if !hasActive {
				r.stopLoopLocked()
				r.mu.Unlock()
				return
			}
			r.mu.Unlock()
		}
	}()
}

// Функция останавливает цикл отрисовки
// Вызывающий код должен уже захватить блокировку r.mu
func (r *TerminalRenderer) stopLoopLocked() {
	r.widgets = nil
	r.lastLineCount = 0
	r.loopRunning = false
}

// Функция отрисовки терминала, вызываемая на каждой итерации цикла отрисовки
// Вызывающий код должен уже захватить блокировку r.mu
func (r *TerminalRenderer) redrawLocked() bool {
	// Поднимаем курсор на кол-во строк равное кол-ву отрисованных на прошлой итерации виджетов
	if r.lastLineCount > 0 {
		fmt.Fprintf(r.out, "\033[%dA", r.lastLineCount)
	}

	// Выводим накопившиеся логи
	for _, message := range r.pendingMessages {
		fmt.Fprintf(r.out, "%s\033[K\n", message)
	}
	r.pendingMessages = r.pendingMessages[:0]

	// Перерисовываем виджеты
	hasActive := false
	for _, widget := range r.widgets {
		content, active := widget.Render()
		fmt.Fprintf(r.out, "%s\033[K\n", content)
		hasActive = hasActive || active
	}
	r.lastLineCount = len(r.widgets)
	return hasActive
}

// Отрисовывает лог
func (r *TerminalRenderer) Log(msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.loopRunning {
		// Нет активных виджетов - пишем напрямую
		fmt.Fprintln(r.out, msg)
		return
	}

	// Есть активные виджеты - буферизуем сообщение до следующей перерисовки
	r.pendingMessages = append(r.pendingMessages, msg)
}

// Дожидается завершения цикла отрисовки, и как следствие вывода всех сообщений
func (r *TerminalRenderer) Wait() {
	r.mu.Lock()
	if !r.loopRunning {
		r.mu.Unlock()
		return
	}
	done := r.done
	r.mu.Unlock()
	<-done
}
