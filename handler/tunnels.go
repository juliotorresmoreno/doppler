package handler

import (
	"io"
	"sync"
)

// tunnelRegistry tracks the connections of open CONNECT and WebSocket tunnels so
// they can be closed together when the service shuts down.
type tunnelRegistry struct {
	mu      sync.Mutex
	closing bool
	conns   map[io.Closer]struct{}
}

var tunnels = &tunnelRegistry{conns: make(map[io.Closer]struct{})}

// track registers conns for a tunnel. The returned function unregisters them.
// It returns ok=false when the service is shutting down and the tunnel must not start.
func (r *tunnelRegistry) track(conns ...io.Closer) (release func(), ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing {
		return nil, false
	}
	for _, c := range conns {
		r.conns[c] = struct{}{}
	}
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, c := range conns {
			delete(r.conns, c)
		}
	}, true
}

// closeAll refuses new tunnels and closes every open one. Closing a connection
// unblocks its relay, which then ends and closes its own sockets.
func (r *tunnelRegistry) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closing = true
	for c := range r.conns {
		c.Close()
	}
	r.conns = make(map[io.Closer]struct{})
}

// reopen allows tunnels again. Used by tests after closeAll.
func (r *tunnelRegistry) reopen() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closing = false
}

// CloseTunnels closes all open tunnels and refuses new ones. Call it when the
// service is shutting down.
func CloseTunnels() {
	tunnels.closeAll()
}
