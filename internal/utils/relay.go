package utils

import (
	"net"
	"sync"
	"time"
)

// Relay copies data both ways between a and b until both directions finish.
//
//   - When one direction reaches EOF, the other side is half-closed (CloseWrite)
//     so the peer sees EOF but can still send its reply.
//   - When both directions are done, both connections are closed.
//   - idle > 0 ends a direction that has had no traffic for that long.
//   - limitKB > 0 throttles each direction to that many KB/s.
//   - maxDuration > 0 closes the tunnel after that long, whatever its activity.
//
// It returns the bytes copied from a to b (up) and from b to a (down).
func Relay(a, b net.Conn, idle time.Duration, limitKB int64, maxDuration time.Duration) (up, down int64) {
	if maxDuration > 0 {
		timer := time.AfterFunc(maxDuration, func() {
			a.Close()
			b.Close()
		})
		defer timer.Stop()
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		up = copyDirection(b, a, idle, limitKB)
	}()
	go func() {
		defer wg.Done()
		down = copyDirection(a, b, idle, limitKB)
	}()
	wg.Wait()
	a.Close()
	b.Close()
	return up, down
}

func copyDirection(dst, src net.Conn, idle time.Duration, limitKB int64) int64 {
	var total int64
	buf := make([]byte, 32*1024)
	for {
		if idle > 0 {
			src.SetReadDeadline(time.Now().Add(idle))
		}
		n, err := src.Read(buf)
		if n > 0 {
			if limitKB > 0 {
				// Hold the average rate: n bytes take n/rate seconds before they are sent.
				time.Sleep(time.Duration(n) * time.Second / time.Duration(limitKB*1024))
			}
			if idle > 0 {
				dst.SetWriteDeadline(time.Now().Add(idle))
			}
			written, werr := dst.Write(buf[:n])
			total += int64(written)
			if werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	closeWrite(dst)
	return total
}

// closeWrite half-closes c when the transport supports it.
func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
}
