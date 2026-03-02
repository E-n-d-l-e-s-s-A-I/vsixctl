package httputil_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/pkg/httputil"
)

func TestProgressReader_Read(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		total          int64
		bufSize        int
		wantCallbacks  int
		wantDownloaded int64
		wantTotal      int64
	}{
		{
			name:           "read_all_at_once",
			data:           []byte("hello"),
			total:          5,
			bufSize:        64,
			wantCallbacks:  2,
			wantDownloaded: 5,
			wantTotal:      5,
		},
		{
			name:           "read_in_chunks",
			data:           []byte("hello world"),
			total:          11,
			bufSize:        3,
			wantCallbacks:  5,
			wantDownloaded: 11,
			wantTotal:      11,
		},
		{
			name:           "empty_reader",
			data:           []byte{},
			total:          0,
			bufSize:        64,
			wantCallbacks:  1,
			wantDownloaded: 0,
			wantTotal:      0,
		},
		{
			name:           "unknown_total",
			data:           []byte("data"),
			total:          -1,
			bufSize:        64,
			wantCallbacks:  2,
			wantDownloaded: 4,
			wantTotal:      -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := io.NopCloser(bytes.NewReader(tt.data))

			var lastDownloaded, lastTotal int64
			callbackCount := 0
			callback := func(downloaded, total int64) {
				lastDownloaded = downloaded
				lastTotal = total
				callbackCount++
			}

			pr := httputil.NewProgressReader(reader, tt.total, callback)

			buf := make([]byte, tt.bufSize)
			var allData []byte
			for {
				n, err := pr.Read(buf)
				if n > 0 {
					allData = append(allData, buf[:n]...)
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if !bytes.Equal(allData, tt.data) {
				t.Errorf("data: got %q, want %q", allData, tt.data)
			}
			if lastDownloaded != tt.wantDownloaded {
				t.Errorf("downloaded: got %d, want %d", lastDownloaded, tt.wantDownloaded)
			}
			if lastTotal != tt.wantTotal {
				t.Errorf("total: got %d, want %d", lastTotal, tt.wantTotal)
			}
			if callbackCount != tt.wantCallbacks {
				t.Errorf("callback count: got %d, want %d", callbackCount, tt.wantCallbacks)
			}
		})
	}
}

// mockCloser - мок для проверки вызова Close
type mockCloser struct {
	io.Reader
	closed bool
	err    error
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.err
}

func TestProgressReader_Close(t *testing.T) {
	t.Run("delegates_close_to_underlying_reader", func(t *testing.T) {
		mock := &mockCloser{Reader: bytes.NewReader([]byte("test"))}
		pr := httputil.NewProgressReader(mock, 4, func(_, _ int64) {})

		err := pr.Close()

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !mock.closed {
			t.Error("Close was not called on underlying reader")
		}
	})

	t.Run("propagates_close_error", func(t *testing.T) {
		wantErr := errors.New("close error")
		mock := &mockCloser{Reader: bytes.NewReader([]byte("test")), err: wantErr}
		pr := httputil.NewProgressReader(mock, 4, func(_, _ int64) {})

		err := pr.Close()

		if !errors.Is(err, wantErr) {
			t.Errorf("error: got %v, want %v", err, wantErr)
		}
	})
}
