//go:build !unix

package prompt

import (
	"errors"
	"io"
	"os"
	"time"
)

func readByteWait(in *os.File, timeout time.Duration) (byte, error) {
	if err := in.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return readOne(in)
	}
	defer in.SetReadDeadline(time.Time{})
	var buf [1]byte
	n, err := in.Read(buf[:])
	if n == 1 {
		return buf[0], nil
	}
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return 0, errKeyTimeout
	}
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, io.EOF
	}
	return 0, err
}
