package cliutils

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Мок виджета для тестирования TerminalRenderer
type mockWidget struct {
	mu      sync.Mutex
	content string
	active  bool
}

func newMockWidget(content string) *mockWidget {
	return &mockWidget{content: content, active: true}
}

func (w *mockWidget) Render() (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.content, w.active
}

func (w *mockWidget) setContent(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.content = s
}

func (w *mockWidget) finish() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.active = false
}

// Проверяет отрисовку одного виджета и его финальное состояние на экране
func TestTerminalRendererSingleWidget(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	w := newMockWidget("ext-a 50/100")
	tr.AddWidget(w)

	w.setContent("ext-a 100/100")
	w.finish()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	if lines[0] != "ext-a 100/100" {
		t.Errorf("got %q, want %q", lines[0], "ext-a 100/100")
	}
}

// Проверяет что несколько виджетов отрисовываются на отдельных строках в правильном порядке
func TestTerminalRendererMultipleWidgets(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	wA := newMockWidget("ext-a 30/100")
	wB := newMockWidget("ext-b 60/200")
	tr.AddWidget(wA)
	tr.AddWidget(wB)

	wA.setContent("ext-a 100/100")
	wB.setContent("ext-b 200/200")
	wA.finish()
	wB.finish()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if lines[0] != "ext-a 100/100" {
		t.Errorf("line 0: got %q, want %q", lines[0], "ext-a 100/100")
	}
	if lines[1] != "ext-b 200/200" {
		t.Errorf("line 1: got %q, want %q", lines[1], "ext-b 200/200")
	}
}

// Проверяет что после завершения всех виджетов цикл перезапускается при добавлении нового
func TestTerminalRendererTickerRestart(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	// Первый цикл: добавляем и завершаем
	w1 := newMockWidget("ext-a done")
	w1.finish()
	tr.AddWidget(w1)
	tr.Wait()

	// Второй цикл: добавляем заново - тикер должен перезапуститься
	w2 := newMockWidget("ext-b 10/50")
	tr.AddWidget(w2)

	w2.setContent("ext-b 50/50")
	w2.finish()
	tr.Wait()

	output := buf.String()
	if !strings.Contains(output, "ext-b 50/50") {
		t.Errorf("expected output to contain %q after ticker restart, got %q", "ext-b 50/50", output)
	}
}

// Проверяет корректность отрисовки при конкурентном обновлении нескольких виджетов
func TestTerminalRendererConcurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	const widgetCount = 10
	var wg sync.WaitGroup
	wg.Add(widgetCount)

	for i := range widgetCount {
		w := newMockWidget(fmt.Sprintf("ext-%d 0/100", i))
		tr.AddWidget(w)
		go func() {
			defer wg.Done()

			for j := range 100 {
				w.setContent(fmt.Sprintf("ext-%d %d/100", i, j+1))
			}
			w.setContent(fmt.Sprintf("ext-%d 100/100", i))
			w.finish()
		}()
	}
	wg.Wait()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != widgetCount {
		t.Fatalf("got %d lines, want %d", len(lines), widgetCount)
	}

	for i, got := range lines {
		want := fmt.Sprintf("ext-%d 100/100", i)
		if got != want {
			t.Errorf("line %d: got %q, want %q", i, got, want)
		}
	}
}

// Проверяет что логи выводятся над виджетами при конкурентной работе
func TestTerminalRendererConcurrentWithLogs(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	const widgetCount = 10
	var wg sync.WaitGroup
	wg.Add(widgetCount)

	widgets := make([]*mockWidget, widgetCount)
	for i := range widgetCount {
		w := newMockWidget(fmt.Sprintf("ext-%d 0/100", i))
		tr.AddWidget(w)
		widgets[i] = w
	}

	for i, w := range widgets {
		go func() {
			defer wg.Done()

			for j := range 100 {
				if j == 4 {
					tr.Log("log")
				}
				w.setContent(fmt.Sprintf("ext-%d %d/100", i, j+1))
			}
			w.setContent(fmt.Sprintf("ext-%d 100/100", i))
		}()
	}
	wg.Wait()

	for _, w := range widgets {
		w.finish()
	}
	tr.Wait()

	lines := renderANSI(buf.String())

	if len(lines) != 2*widgetCount {
		t.Fatalf("got %d lines, want %d", len(lines), 2*widgetCount)
	}

	// Проверяем что наверху логи
	for i, got := range lines[:widgetCount] {
		if got != "log" {
			t.Errorf("log line %d: got %q, want %q", i, got, "log")
		}
	}

	// Проверяем что снизу виджеты
	for i, got := range lines[widgetCount:] {
		want := fmt.Sprintf("ext-%d 100/100", i)
		if got != want {
			t.Errorf("widget line %d: got %q, want %q", i, got, want)
		}
	}
}

// Проверяет что Log без активных виджетов пишет напрямую в out
func TestTerminalRendererLogWithoutWidgets(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	tr.Log("hello")
	tr.Log("world")

	got := buf.String()
	want := "hello\nworld\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Проверяет что лог, отправленный между добавлением виджетов, отображается над ними
func TestTerminalRendererLogBetweenWidgetsAdd(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	wA := newMockWidget("ext-a 30/100")
	tr.AddWidget(wA)
	wA.setContent("ext-a 100/100")

	tr.Log("log")

	wB := newMockWidget("ext-b 60/200")
	tr.AddWidget(wB)
	wB.setContent("ext-b 200/200")

	wA.finish()
	wB.finish()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	if lines[0] != "log" {
		t.Errorf("line 0: got %q, want %q", lines[0], "log")
	}
	if lines[1] != "ext-a 100/100" {
		t.Errorf("line 1: got %q, want %q", lines[1], "ext-a 100/100")
	}
	if lines[2] != "ext-b 200/200" {
		t.Errorf("line 2: got %q, want %q", lines[2], "ext-b 200/200")
	}
}

// Проверяет что лог, отправленный между запусками цикла, оказывается между виджетами
func TestTerminalRendererLogBetweenCycleRestart(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 0)

	wA := newMockWidget("ext-a 30/100")
	tr.AddWidget(wA)
	wA.setContent("ext-a 100/100")
	wA.finish()

	wB := newMockWidget("ext-b 60/200")
	tr.AddWidget(wB)
	wB.setContent("ext-b 200/200")
	wB.finish()
	tr.Wait()

	tr.Log("log")

	wC := newMockWidget("ext-c 60/300")
	tr.AddWidget(wC)
	wC.setContent("ext-c 300/300")
	wC.finish()

	wD := newMockWidget("ext-d 60/400")
	tr.AddWidget(wD)
	wD.setContent("ext-d 400/400")
	wD.finish()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	if lines[0] != "ext-a 100/100" {
		t.Errorf("line 0: got %q, want %q", lines[0], "ext-a 100/100")
	}
	if lines[1] != "ext-b 200/200" {
		t.Errorf("line 1: got %q, want %q", lines[1], "ext-b 200/200")
	}
	if lines[2] != "log" {
		t.Errorf("line 2: got %q, want %q", lines[2], "log")
	}
	if lines[3] != "ext-c 300/300" {
		t.Errorf("line 3: got %q, want %q", lines[3], "ext-c 300/300")
	}
	if lines[4] != "ext-d 400/400" {
		t.Errorf("line 4: got %q, want %q", lines[4], "ext-d 400/400")
	}
}

// Проверяет что виджеты с длинным содержимым обрезаются до terminalWidth
func TestTerminalRendererTruncatesLongLines(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := NewTerminalRenderer(buf, time.Millisecond, 20)

	w := newMockWidget(strings.Repeat("x", 50))
	tr.AddWidget(w)
	w.finish()
	tr.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	if len(lines[0]) != 20 {
		t.Errorf("line length: got %d, want 20", len(lines[0]))
	}
	if lines[0] != strings.Repeat("x", 20) {
		t.Errorf("got %q, want %q", lines[0], strings.Repeat("x", 20))
	}
}

// renderANSI интерпретирует ANSI-вывод и возвращает финальное состояние экрана.
// Поддерживает \033[nA (курсор вверх на n строк) и \033[K (очистка строки).
func renderANSI(raw string) []string {
	ansiPattern := regexp.MustCompile(`\033\[(\d+)A|\033\[K`)

	var screen []string
	cursor := 0

	parts := ansiPattern.Split(raw, -1)
	matches := ansiPattern.FindAllStringSubmatch(raw, -1)

	matchIdx := 0
	for i, part := range parts {
		if part != "" {
			textLines := strings.Split(part, "\n")
			for j, line := range textLines {
				if j > 0 {
					cursor++
				}
				if line == "" {
					continue
				}
				for cursor >= len(screen) {
					screen = append(screen, "")
				}
				screen[cursor] = line
			}
		}

		if i < len(parts)-1 && matchIdx < len(matches) {
			match := matches[matchIdx]
			matchIdx++
			if match[1] != "" {
				n, _ := strconv.Atoi(match[1])
				cursor -= n
				if cursor < 0 {
					cursor = 0
				}
			}
		}
	}

	for len(screen) > 0 && screen[len(screen)-1] == "" {
		screen = screen[:len(screen)-1]
	}

	return screen
}
