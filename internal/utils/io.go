package utils

import (
	"bytes"
	"io"
)

type WriteCloserLogger struct {
	io.WriteCloser
	buffer *bytes.Buffer
}

func (el WriteCloserLogger) Write(p []byte) (n int, err error) {
	return el.WriteCloser.Write(p)
}

func (el WriteCloserLogger) Close() error {
	return el.WriteCloser.Close()
}

func Transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()

	d := WriteCloserLogger{
		WriteCloser: destination,
		buffer:      bytes.NewBufferString(""),
	}

	io.Copy(d, source)
}
