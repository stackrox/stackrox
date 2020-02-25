package ioutils

import "io"

// ReadAtMost reads at most the given number of bytes from the reader. The returned buffer is at most num bytes long and
// is guaranteed to contain exactly the bytes that were actually read, even if an error is encountered. No EOF
// conditions are ever returned, even if the reader reported EOF before the first byte was read.
func ReadAtMost(r io.Reader, num int) ([]byte, error) {
	buf := make([]byte, num)
	n, err := io.ReadFull(r, buf)
	buf = buf[:n]
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	return buf, err
}
