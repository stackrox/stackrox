// Package pulp is for reading a Pulp manifest.
package pulp

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
)

// A Manifest is a series of Entries indicating where to find resources in the
// pulp repository.
type Manifest []Entry

// Entry is an entry in a pulp manifest.
type Entry struct {
	// This path should be parsed in the context of the manifest's URL.
	Path     string
	Checksum []byte
	Size     int64
}

// Load populates the manifest from the io.Reader.
//
// The data is expected in the manifest CSV format.
func (m *Manifest) Load(r io.Reader) error {
	rd := csv.NewReader(r)
	rd.FieldsPerRecord = 3
	rd.ReuseRecord = true
	l := 0
	rec, err := rd.Read()
	for ; err == nil; rec, err = rd.Read() {
		var err error
		e := Entry{}
		e.Path += rec[0] // This += should result in us getting a copy.
		e.Checksum, err = hex.DecodeString(rec[1])
		if err != nil {
			return fmt.Errorf("line %d: %w", l, err)
		}
		e.Size, err = strconv.ParseInt(rec[2], 10, 64)
		if err != nil {
			return fmt.Errorf("line %d: %w", l, err)
		}
		*m = append(*m, e)
		l++
	}
	if err != io.EOF {
		return err
	}
	return nil
}
