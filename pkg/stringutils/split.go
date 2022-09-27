package stringutils

import "strings"

// Split2 splits the given string at the given separator, returning the part before and after the separator as two
// separate return values.
// If the string does not contain `sep`, the entire string is returned as the first return value.
func Split2(str string, sep string) (string, string) {
	splitIdx := strings.Index(str, sep)
	if splitIdx == -1 {
		return str, ""
	}
	return str[:splitIdx], str[splitIdx+len(sep):]
}

// Split2Last splits the given string at the last instance of the given separator,
// returning the part before and after the separator as two separate return values.
// If the string does not contain `sep`, the entire string is returned as the first return value.
func Split2Last(str string, sep string) (string, string) {
	splitIdx := strings.LastIndex(str, sep)
	if splitIdx == -1 {
		return str, ""
	}
	return str[:splitIdx], str[splitIdx+len(sep):]
}

// SplitNPadded acts like `strings.SplitN`, but will *always* return a slice of length n (padded with empty strings
// if necessary).
func SplitNPadded(str string, sep string, n int) []string {
	res := strings.SplitN(str, sep, n)
	for len(res) < n {
		res = append(res, "")
	}
	return res
}

// GetUpTo gets the values up to the separator or returns the input string if the separator does not exist
func GetUpTo(str, sep string) string {
	part1, _ := Split2(str, sep)
	return part1
}

// GetAfter gets the substring after the separator or returns the input string if the separator does not exist
func GetAfter(str, sep string) string {
	p1, p2 := Split2(str, sep)
	if len(p1) == len(str) {
		return p1
	}
	return p2
}

// GetAfterLast gets the substring after the last instance of the given separator
// or returns the input string if the separator does not exist
func GetAfterLast(str, sep string) string {
	p1, p2 := Split2Last(str, sep)
	if len(p1) == len(str) && sep != "" {
		return p1
	}
	return p2
}

// GetBetween gets the string between the two passed strings, otherwise it returns the empty string
func GetBetween(str, start, end string) string {
	startIdx := strings.Index(str, start)
	if startIdx == -1 || startIdx == len(str)-len(start) {
		return ""
	}
	offset := startIdx + len(start)
	endIdx := strings.Index(str[offset:], end)
	if endIdx == -1 {
		return ""
	}
	return str[offset : offset+endIdx]
}
