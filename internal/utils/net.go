package utils

import (
	"io"
	"time"
)

func CopyWithRateLimit(dst io.Writer, src io.Reader, rateLimit int64) (int64, time.Duration, error) {
	buffer := make([]byte, 4096)

	var totalBytesCopied int64
	startTime := time.Now()

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			totalBytesCopied += int64(n)
			_, err := dst.Write(buffer[:n])
			if err != nil {
				return totalBytesCopied, time.Since(startTime), err
			}
		}

		// No hay suficientes tokens, calcular el tiempo de espera basado en la velocidad
		waitTime := time.Microsecond * time.Duration(rateLimit)
		time.Sleep(waitTime)

		if err == io.EOF {
			break
		} else if err != nil {
			return totalBytesCopied, time.Since(startTime), err
		}
	}

	elapsedTime := time.Since(startTime)
	return totalBytesCopied, elapsedTime, nil
}
