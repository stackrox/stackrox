package htpasswd

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/crypto/bcrypt"
)

const (
	// This is the character between the username and the hash.
	delimiter rune = ':'
)

var (
	log = logging.LoggerForModule()
)

// A HashFile can read or write the format produced by the Apache `htpasswd`
// program and validate passwords against the contents of such a file.
// Only bcrypt entries are supported for security reasons.
// See https://httpd.apache.org/docs/2.4/misc/password_encryptions.html.
type HashFile struct {
	hashes map[username]hash
}

type username string

type hash []byte

// New creates a new, empty registry of users and password hashes.
func New() *HashFile {
	return &HashFile{
		hashes: make(map[username]hash),
	}
}

// ReadHashFile loads a HashFile from htpasswd format.
func ReadHashFile(r io.Reader) (*HashFile, error) {
	c := csv.NewReader(r)
	c.Comma = delimiter
	c.Comment = '#'
	c.TrimLeadingSpace = true

	entries, err := c.ReadAll()
	if err != nil {
		return nil, err
	}

	hashes := make(map[username]hash)
	for i, entry := range entries {
		if len(entry) < 2 {
			log.Warnf("Invalid hash on line %d: %s", i, err)
			continue
		}

		user := username(entry[0])

		tenant, uname, found := strings.Cut(entry[0], "-")

		if !found || len(tenant) == 0 || len(uname) == 0 {
			log.Warnf("Invalid username: %s", user)
			continue
		}

		userHash := hash(entry[1])

		// Check validity of the hash by verifying it can be parsed.
		_, err := bcrypt.Cost(userHash)
		if err != nil {
			log.Warnf("Invalid hash on line %d: %s", i, err)
			continue
		}
		hashes[user] = userHash
	}

	return &HashFile{
		hashes: hashes,
	}, nil
}

// Check compares the provided password with the known password for the user.
func (hf *HashFile) Check(user, pass string) bool {
	if hf == nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hf.hashes[username(user)]), []byte(pass)) == nil
}
