package utils

import (
	"bytes"
	"net"
)

type ConnLogger struct {
	net.Conn
	Buffer *bytes.Buffer
}

func (el ConnLogger) Write(b []byte) (int, error) {
	if el.Buffer != nil {
		el.Buffer.Write(b)
	}
	return el.Conn.Write(b)
}
