package utils

import "iter"

type indents []int

func (ptr *indents) pop() int {
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

// words yields words, spaces and new lines.
func words(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		begin := 0
		for end, ch := range s {
			if ch != '\n' && ch != ' ' && ch != '\t' {
				continue
			}
			if end > begin {
				if !yield(s[begin:end]) {
					return
				}
				begin = end
			}
			if !yield(string(ch)) {
				return
			}
			begin++
		}
		if begin < len(s) {
			yield(s[begin:])
		}
	}
}
