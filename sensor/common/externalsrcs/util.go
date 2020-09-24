package externalsrcs

// `sortableIPv4NetworkSlice` and `sortableIPv6NetworkSlice` allows us to sort the networks in descending order of
// prefix length. Networks with same prefix length are ordered descending lexical byte order. Since, the host identifier
// bits (bits not in network prefix) are all set to 0, this gives us smallest to largest subnet ordering.
// e.g. 127.0.0.0/8, 10.10.0.0/24, 10.0.0.0/24, 10.0.0.0/8...
type sortableIPv4NetworkSlice []byte

func (s sortableIPv4NetworkSlice) Len() int {
	return len(s) / 5
}

func (s sortableIPv4NetworkSlice) Less(i, j int) bool {
	for k := 0; k < 5; k++ {
		if s[5*i+k] != s[5*j+k] {
			return s[5*i+k] > s[5*j+k]
		}
	}
	return false
}

func (s sortableIPv4NetworkSlice) Swap(i, j int) {
	for k := 0; k < 5; k++ {
		s[5*i+k], s[5*j+k] = s[5*j+k], s[5*i+k]
	}
}

type sortableIPv6NetworkSlice []byte

func (s sortableIPv6NetworkSlice) Len() int {
	return len(s) / 17
}

func (s sortableIPv6NetworkSlice) Less(i, j int) bool {
	for k := 0; k < 17; k++ {
		if s[17*i+k] != s[17*j+k] {
			return s[17*i+k] > s[17*j+k]
		}
	}
	return false
}

func (s sortableIPv6NetworkSlice) Swap(i, j int) {
	for k := 0; k < 17; k++ {
		s[17*i+k], s[17*j+k] = s[17*j+k], s[17*i+k]
	}
}
