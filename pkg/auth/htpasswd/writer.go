package htpasswd

import (
	"encoding/csv"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcryptCost sets the difficulty of the bcrypt hash function,
	// which affects the time required to compute a hash.
	bcryptCost = 5
)

// Write outputs the current configured users and hashes to the provided writer.
func (hf *HashFile) Write(w io.Writer) error {
	entries := make([][]string, 0, len(hf.hashes))
	for u, h := range hf.hashes {
		entries = append(entries, []string{string(u), string(h)})
	}

	c := csv.NewWriter(w)
	c.Comma = delimiter
	return c.WriteAll(entries)
}

// Set emplaces a hash of the provided password for the provided user.
// Updates will be reflected in validation and in output.
func (hf *HashFile) Set(user, pass string) error {
	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcryptCost)
	if err != nil {
		return errors.Wrap(err, "hash")
	}
	hf.hashes[username(user)] = passHash
	return nil
}
