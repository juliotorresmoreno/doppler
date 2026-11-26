package helper

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
)

func GetResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	metaValue := reflect.ValueOf(w).Elem()
	field := metaValue.FieldByName("ResponseWriter")
	if field.IsValid() && field.Interface() != nil {
		value := field.Interface()
		return value.(http.ResponseWriter)
	}
	return w
}

// GetHijack takes over the connection. Bytes the server already read past the
// request stay in its buffer, so the returned connection reads from that buffer
// first; otherwise a client that sends data right after CONNECT would lose it.
func GetHijack(w http.ResponseWriter) (net.Conn, error) {
	sw := GetResponseWriter(w)
	hijacker, ok := sw.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("%v", "Hijacking not supported")
	}

	c, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}
	return NewBufferedConn(c, rw.Reader), nil
}

// BufferedConn reads from r (which may hold bytes already consumed from the
// underlying connection) and writes to the connection directly.
type BufferedConn struct {
	net.Conn
	r io.Reader
}

func NewBufferedConn(c net.Conn, r io.Reader) net.Conn {
	return &BufferedConn{Conn: c, r: r}
}

func (b *BufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}
