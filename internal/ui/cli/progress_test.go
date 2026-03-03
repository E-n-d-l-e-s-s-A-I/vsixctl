package cli

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

// записанный вызов Draw
type drawCall struct {
	label      string
	downloaded int64
	total      int64
}

// мок, записывающий все вызовы Draw
type spyProgressBar struct {
	mu    sync.Mutex
	calls []drawCall
}

func (s *spyProgressBar) Draw(label string, downloaded, total int64) string {
	s.mu.Lock()
	s.calls = append(s.calls, drawCall{label, downloaded, total})
	s.mu.Unlock()
	return fmt.Sprintf("%s %d/%d", label, downloaded, total)
}

// lastCallFor возвращает последний вызов Draw для указанного label
func (s *spyProgressBar) lastCallFor(label string) (drawCall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.calls) - 1; i >= 0; i-- {
		if s.calls[i].label == label {
			return s.calls[i], true
		}
	}
	return drawCall{}, false
}

func TestProgressManagerSingleBar(t *testing.T) {
	buf := bytes.Buffer{}
	spy := &spyProgressBar{}
	pm := NewProgressManager(&buf, time.Millisecond, spy)

	onProgress, finish := pm.AddBar("ext-a")
	onProgress(50, 100)

	// Ждём чтобы тикер успел отрисовать
	time.Sleep(5 * time.Millisecond)

	call, ok := spy.lastCallFor("ext-a")
	if !ok {
		t.Fatal("expected Draw call for ext-a, got none")
	}
	if call.downloaded != 50 || call.total != 100 {
		t.Errorf("got %d/%d, want 50/100", call.downloaded, call.total)
	}

	onProgress(100, 100)
	finish()

	call, ok = spy.lastCallFor("ext-a")
	if !ok {
		t.Fatal("expected Draw call after finish, got none")
	}
	if call.downloaded != 100 || call.total != 100 {
		t.Errorf("got %d/%d, want 100/100", call.downloaded, call.total)
	}
}

func TestProgressManagerMultipleBars(t *testing.T) {
	buf := bytes.Buffer{}
	spy := &spyProgressBar{}
	pm := NewProgressManager(&buf, time.Millisecond, spy)

	onProgressA, finishA := pm.AddBar("ext-a")
	onProgressB, finishB := pm.AddBar("ext-b")

	onProgressA(30, 100)
	onProgressB(60, 200)

	// Ждём чтобы тикер успел отрисовать оба бара
	time.Sleep(5 * time.Millisecond)

	callA, ok := spy.lastCallFor("ext-a")
	if !ok {
		t.Fatal("expected Draw call for ext-a, got none")
	}
	if callA.downloaded != 30 || callA.total != 100 {
		t.Errorf("ext-a: got %d/%d, want 30/100", callA.downloaded, callA.total)
	}

	callB, ok := spy.lastCallFor("ext-b")
	if !ok {
		t.Fatal("expected Draw call for ext-b, got none")
	}
	if callB.downloaded != 60 || callB.total != 200 {
		t.Errorf("ext-b: got %d/%d, want 60/200", callB.downloaded, callB.total)
	}

	finishA()
	finishB()
}

func TestProgressManagerFinishRedraw(t *testing.T) {
	buf := bytes.Buffer{}
	spy := &spyProgressBar{}
	// Длинный интервал - тикер не успеет сработать
	pm := NewProgressManager(&buf, time.Hour, spy)

	onProgress, finish := pm.AddBar("ext-a")
	onProgress(100, 100)
	finish()

	// finish вызывает redrawLocked синхронно, без тикера
	call, ok := spy.lastCallFor("ext-a")
	if !ok {
		t.Fatal("expected Draw call after finish, got none")
	}
	if call.downloaded != 100 || call.total != 100 {
		t.Errorf("got %d/%d, want 100/100", call.downloaded, call.total)
	}
}

func TestProgressManagerTickerRestart(t *testing.T) {
	buf := bytes.Buffer{}
	spy := &spyProgressBar{}
	pm := NewProgressManager(&buf, time.Millisecond, spy)

	// Первый цикл: добавляем и завершаем
	_, finish := pm.AddBar("ext-a")
	finish()

	callsBefore := len(spy.calls)

	// Второй цикл: добавляем заново - тикер должен перезапуститься
	onProgress, finish2 := pm.AddBar("ext-b")
	onProgress(10, 50)
	time.Sleep(5 * time.Millisecond)

	call, ok := spy.lastCallFor("ext-b")
	if !ok {
		t.Fatal("expected Draw call for ext-b after ticker restart, got none")
	}
	if call.downloaded != 10 || call.total != 50 {
		t.Errorf("got %d/%d, want 10/50", call.downloaded, call.total)
	}

	finish2()

	if len(spy.calls) <= callsBefore {
		t.Error("expected new Draw calls after ticker restart, got none")
	}
}

// Запускаем асинхронно несколько прогресс баров
// и проверяем конечное состояние экрана out
func TestProgressManagerConcurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	spy := &spyProgressBar{}
	pm := NewProgressManager(buf, time.Millisecond, spy)

	const barCount = 10
	var wg sync.WaitGroup
	wg.Add(barCount)

	for i := range barCount {
		label := fmt.Sprintf("ext-%d", i)
		onProgress, finish := pm.AddBar(label)
		go func() {
			defer wg.Done()

			for j := range 100 {
				onProgress(int64(j+1), 100)
			}
			finish()
		}()
	}
	wg.Wait()

	lines := renderANSI(buf.String())
	if len(lines) != barCount {
		t.Fatalf("got %d count of lines, want %d count of lines", len(lines), barCount)
	}

	for i, got := range lines {
		want := fmt.Sprintf("ext-%d 100/100", i)
		if got != want {
			t.Errorf("line %d: got %q, want %q", i, got, want)
		}
	}
}

// renderANSI интерпретирует ANSI-вывод и возвращает финальное состояние экрана.
// Поддерживает \033[nA (курсор вверх на n строк) и \033[K (очистка строки).
func renderANSI(raw string) []string {
	// Разбиваем вывод на токены: ANSI-коды и обычный текст
	ansiPattern := regexp.MustCompile(`\033\[(\d+)A|\033\[K`)

	var screen []string
	cursor := 0

	// Разбиваем на строки по \n, но сначала обрабатываем ANSI-коды
	parts := ansiPattern.Split(raw, -1)
	matches := ansiPattern.FindAllStringSubmatch(raw, -1)

	matchIdx := 0
	for i, part := range parts {
		// Записываем текст на текущую позицию курсора
		if part != "" {
			textLines := strings.Split(part, "\n")
			for j, line := range textLines {
				if j > 0 {
					cursor++
				}
				if line == "" {
					continue
				}
				// Расширяем экран при необходимости
				for cursor >= len(screen) {
					screen = append(screen, "")
				}
				screen[cursor] = line
			}
		}

		// Применяем ANSI-код после этого фрагмента текста
		if i < len(parts)-1 && matchIdx < len(matches) {
			match := matches[matchIdx]
			matchIdx++
			if match[1] != "" {
				// \033[nA - курсор вверх
				n, _ := strconv.Atoi(match[1])
				cursor -= n
				if cursor < 0 {
					cursor = 0
				}
			}
			// \033[K - очистка строки (просто перезапишется при следующей записи)
		}
	}

	// Убираем пустые строки в конце
	for len(screen) > 0 && screen[len(screen)-1] == "" {
		screen = screen[:len(screen)-1]
	}

	return screen
}
