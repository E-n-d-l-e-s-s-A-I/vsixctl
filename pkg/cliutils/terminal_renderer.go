package cliutils

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Интерфейс виджета, с которыми работает TerminalRenderer
type Widget interface {
	// Рендер виджета, возвращает:
	// string - содержимое виджета,
	// bool - нужны ли ещё отрисовки:
	//   true - виджет еще нужно рисовать
	//   false - виджет больше не нужно рисовать
	Render(termWidth int) (string, bool)
}

// Полностью отвечает за вывод в терминал, нужен чтобы синхронизировать запись в поток вывода
// Поддерживает как динамические виджеты, которые должны реализовывать интерфейс Widget
// Так и вывод обычных сообщений
type TerminalRenderer struct {
	out                io.Writer     // Поток вывода
	outWidth           func() int    // Функция возвращающая ширину потока. Инжектим её, чтобы избежать зависимости от конкретного потока вывода
	redrawInterval     time.Duration // Интервал отрисовки
	widgets            []Widget      // Слайс отслеживаемых виджетов
	pendingMessages    []string      // Очередь сообщений(логов) для отрисовки
	loopRunning        bool          // Запущен ли цикл отрисовки
	lastContentLengths []int         // Длины контента виджетов с прошлой итерации (для подсчёта визуальных строк)
	mu                 sync.Mutex    // Мьютекс для синхронизации доступа к общему состоянию
	done               chan struct{} // Канал завершения
}

func NewTerminalRenderer(out io.Writer, outWidth func() int, interval time.Duration) *TerminalRenderer {
	return &TerminalRenderer{
		out:            out,
		outWidth:       outWidth,
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
	r.lastContentLengths = nil
	r.loopRunning = false
}

// Считает кол-во визуальных строк, которые контент длиной contentLen
// занимает при ширине терминала termWidth
func visualLineCount(contentLen, termWidth int) int {
	if termWidth <= 0 || contentLen <= termWidth {
		return 1
	}
	return (contentLen + termWidth - 1) / termWidth
}

// Функция отрисовки терминала, вызываемая на каждой итерации цикла отрисовки
// Вызывающий код должен уже захватить блокировку r.mu
func (r *TerminalRenderer) redrawLocked() bool {
	// Получаем текущую ширину терминала
	termWidth := r.outWidth()

	// Считаем сколько визуальных строк заняли виджеты на прошлой итерации
	// с учётом текущей ширины терминала (пользователь мог изменить размер окна)
	prevVisualLines := 0
	for _, contentLen := range r.lastContentLengths {
		prevVisualLines += visualLineCount(contentLen, termWidth)
	}

	// Поднимаем курсор на кол-во визуальных строк с прошлой итерации
	if prevVisualLines > 0 {
		fmt.Fprintf(r.out, "\033[%dA", prevVisualLines)
	}

	// Выводим накопившиеся логи
	for _, message := range r.pendingMessages {
		fmt.Fprintf(r.out, "%s\033[K\n", message)
	}
	r.pendingMessages = r.pendingMessages[:0]

	// Перерисовываем виджеты и запоминаем длины контента
	r.lastContentLengths = r.lastContentLengths[:0]
	hasActive := false
	for _, widget := range r.widgets {
		content, active := widget.Render(termWidth)
		if termWidth > 0 && len(content) > termWidth {
			content = content[:termWidth]
		}
		r.lastContentLengths = append(r.lastContentLengths, len(content))
		fmt.Fprintf(r.out, "%s\033[K\n", content)
		hasActive = hasActive || active
	}

	// Затираем висячие строки, оставшиеся от прошлой итерации
	// (при сужении терминала прошлые строки занимали больше визуальных строк)
	staleLines := prevVisualLines - len(r.widgets)
	if staleLines > 0 {
		for i := 0; i < staleLines; i++ {
			fmt.Fprintf(r.out, "\033[K\n")
		}
		// Возвращаем курсор обратно к концу виджетов,
		// чтобы на следующей итерации cursor-up поднялся ровно на len(widgets)
		fmt.Fprintf(r.out, "\033[%dA", staleLines)
	}

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
