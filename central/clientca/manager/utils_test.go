package manager

import (
	"strconv"
	"testing"
)

func TestFormatID(t *testing.T) {
	testcases := []struct {
		subject   []byte
		formatted string
	}{
		{
			subject:   []byte{},
			formatted: "",
		},
		{
			subject:   []byte{0x1},
			formatted: "01",
		},
		{
			subject:   []byte{0xa0, 0x1},
			formatted: "A0:01",
		},
		{
			subject:   []byte{0x1, 0x10, 0xa0},
			formatted: "01:10:A0",
		},
	}
	for i, tc := range testcases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			val := formatID(tc.subject)
			if val != tc.formatted {
				t.Errorf("Expected %q but got %q", tc.formatted, val)
			}
		})
	}
}
