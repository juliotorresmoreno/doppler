package utils

import (
	"io"
	"time"
)

func CopyWithRateLimit(dst io.Writer, src io.Reader, rateKB int64) (int64, time.Duration, error) {
	buf := make([]byte, 32*1024)
	rateBytes := rateKB * 1024

	var copied int64
	start := time.Now()
	windowStart := start
	var windowBytes int64

	for {
		n, err := src.Read(buf)
		if n > 0 {
			written, werr := dst.Write(buf[:n])
			if werr != nil {
				return copied, time.Since(start), werr
			}

			copied += int64(written)
			windowBytes += int64(written)

			elapsed := time.Since(windowStart)
			expected := time.Duration(windowBytes*int64(time.Second)) / time.Duration(rateBytes)

			if expected > elapsed {
				time.Sleep(expected - elapsed)
			}

			if elapsed >= time.Second {
				windowStart = time.Now()
				windowBytes = 0
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return copied, time.Since(start), err
		}
	}

	return copied, time.Since(start), nil
}
