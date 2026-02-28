package httputil

import "io"

type ProgressReader struct {
	reader   io.ReadCloser
	total    int64
	download int64
	callback func(downloaded, total int64)
}

func NewProgressReader(reader io.ReadCloser, total int64, callback func(downloaded, total int64)) *ProgressReader {
	return &ProgressReader{
		reader:   reader,
		total:    total,
		callback: callback,
	}
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	read, err := pr.reader.Read(p)
	pr.download += int64(read)
	pr.callback(pr.download, pr.total)
	return read, err
}

func (pr *ProgressReader) Close() error {
	return pr.reader.Close()
}
