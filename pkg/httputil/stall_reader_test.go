package httputil_test

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/httputil"
)

func TestStallReader_Read(t *testing.T) {
	t.Run("normal_read", func(t *testing.T) {
		data := []byte("hello world")
		reader := io.NopCloser(bytes.NewReader(data))
		sr := httputil.NewStallReader(reader, time.Second)

		got, err := io.ReadAll(sr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("got %q, want %q", got, data)
		}
	})

	t.Run("stall_detected", func(t *testing.T) {
		// blockingReader блокирует Read навсегда
		reader := newBlockingReader()
		sr := httputil.NewStallReader(reader, 50*time.Millisecond)

		_, err := sr.Read(make([]byte, 64))

		if !errors.Is(err, httputil.ErrStalled) {
			t.Errorf("got %v, want %v", err, httputil.ErrStalled)
		}
		if !reader.closed {
			t.Error("underlying reader was not closed on stall")
		}
	})

	t.Run("propagates_read_error", func(t *testing.T) {
		wantErr := errors.New("connection reset")
		reader := &errorReader{err: wantErr}
		sr := httputil.NewStallReader(reader, time.Second)

		_, err := sr.Read(make([]byte, 64))

		if !errors.Is(err, wantErr) {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}

func TestStallReader_Close(t *testing.T) {
	reader := io.NopCloser(bytes.NewReader([]byte("test")))
	sr := httputil.NewStallReader(reader, time.Second)

	err := sr.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// blockingReader блокирует Read до вызова Close
type blockingReader struct {
	ch     chan struct{}
	closed bool
}

func newBlockingReader() *blockingReader {
	return &blockingReader{ch: make(chan struct{})}
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.ch
	return 0, errors.New("closed")
}

func (r *blockingReader) Close() error {
	r.closed = true
	select {
	case <-r.ch:
	default:
		close(r.ch)
	}
	return nil
}

// errorReader сразу возвращает ошибку
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

func (r *errorReader) Close() error {
	return nil
}
