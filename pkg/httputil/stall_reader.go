package httputil

import (
	"errors"
	"io"
	"time"
)

type readResult struct {
	read int
	err  error
}

var ErrStalled = errors.New("stall Reader timeout")

// Обёртка над io.ReadCloser, возвращающая ошибку если данные не поступают в течение timeout
type StallReader struct {
	reader  io.ReadCloser
	timeout time.Duration
}

func NewStallReader(reader io.ReadCloser, timeout time.Duration) *StallReader {
	return &StallReader{reader, timeout}

}

func (sr *StallReader) Read(p []byte) (int, error) {
	// Создаем промежуточный буфер, чтобы избежать data race
	// В ситуации когда истёк таймаут, мы вернули ошибку
	// Но горутина sr.reader.Read(p) ещё не завершилась и может писать в p
	buf := make([]byte, len(p))

	ch := make(chan readResult, 1)
	go func() {
		n, err := sr.reader.Read(buf)
		ch <- readResult{n, err}
	}()

	select {
	case res := <-ch:
		copy(p, buf[:res.read])
		return res.read, res.err
	case <-time.After(sr.timeout):
		sr.reader.Close()
		return 0, ErrStalled
	}
}

func (sr *StallReader) Close() error {
	return sr.reader.Close()
}
