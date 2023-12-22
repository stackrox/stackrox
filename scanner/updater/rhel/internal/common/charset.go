package common

import (
	"io"

	"golang.org/x/text/encoding/ianaindex"
)

// CharsetReader is a function suitable for using as an xml.Decoder's
// CharsetReader member.
func CharsetReader(charset string, r io.Reader) (io.Reader, error) {
	// equivalence hacks
	switch charset {
	case "ASCII":
		charset = `US-ASCII`
	}
	enc, err := ianaindex.IANA.Encoding(charset)
	if err != nil {
		return nil, err
	}
	return enc.NewDecoder().Reader(r), nil
}
