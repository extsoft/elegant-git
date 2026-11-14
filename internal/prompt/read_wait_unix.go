//go:build unix

package prompt

import (
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func readByteWait(in *os.File, timeout time.Duration) (byte, error) {
	fd := int(in.Fd())
	pfd := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	ms := int(timeout.Milliseconds())
	for {
		n, err := unix.Poll(pfd, ms)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, errKeyTimeout
		}
		var buf [1]byte
		nread, err := in.Read(buf[:])
		if nread == 1 {
			return buf[0], nil
		}
		if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, io.EOF
		}
		return 0, err
	}
}
