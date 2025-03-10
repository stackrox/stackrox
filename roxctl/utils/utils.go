package utils

import (
	"strings"
)

type indents []int

func (ptr *indents) popNotLast() int {
	slice := *ptr
	switch len(slice) {
	case 0:
		return 0
	case 1:
		return slice[0]
	default:
		value := slice[0]
		*ptr = slice[1:]
		return value
	}
}

// wordsAndDelimeters is a bufio.SplitFunc function that yields words and
// delimiters.
func wordsAndDelimeters(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := strings.IndexAny(string(data), " \n\t"); i != -1 {
		i = max(i, 1) // i==0 if data starts with a delimeter.
		return i, data[:i], nil
	}
	return len(data), data, nil
}
