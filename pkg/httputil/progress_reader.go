package httputil

import "io"

// Обёртка над io.ReadCloser вызывающая при каждом чтении колбек
type ProgressReader struct {
	reader   io.ReadCloser
	download int64
	callback func(downloaded int64)
}

func NewProgressReader(reader io.ReadCloser, callback func(downloaded int64)) *ProgressReader {
	return &ProgressReader{
		reader:   reader,
		callback: callback,
	}
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	read, err := pr.reader.Read(p)
	pr.download += int64(read)
	pr.callback(pr.download)
	return read, err
}

func (pr *ProgressReader) Close() error {
	return pr.reader.Close()
}
