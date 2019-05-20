package manager

import (
	"encoding/hex"
	"fmt"
	"os"
)

func upper(b byte) byte {
	if b < 'a' || b > 'z' {
		return b
	}
	return b - 'a' + 'A'
}

// ref: https://codereview.stackexchange.com/a/165708
func formatID(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	out := make([]byte, 0, len(b)*3)
	x := out[1*len(b) : 3*len(b)]
	hex.Encode(x, b)
	fmt.Fprintf(os.Stderr, "%s\n", x)
	for i := 0; i < len(x); i += 2 {
		out = append(out, upper(x[i]), upper(x[i+1]), ':')
	}
	return string(out[:len(out)-1])
}
