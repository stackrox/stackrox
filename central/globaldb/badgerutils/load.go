package badgerutils

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

const (
	// MagicNumber is the value of the magic at the beginning of the new backups
	MagicNumber uint32 = 0x42444752
)

// Load wraps Badger load and switches correctly based on the version
func Load(r io.Reader, db *badger.DB) error {
	bufferedReader := bufio.NewReader(r)

	firstBytes, err := bufferedReader.Peek(8)

	if err != nil && err != io.EOF {
		return err
	} else if err == io.EOF {
		return io.ErrUnexpectedEOF
	}

	// Default to legacy backup, but check first 8 bytes to see if there is the magic number
	// and the backup version
	backupVersion := uint32(1)
	if binary.BigEndian.Uint32(firstBytes[:4]) == MagicNumber {
		backupVersion = binary.BigEndian.Uint32(firstBytes[4:8])
		if _, err := bufferedReader.Discard(8); err != nil {
			return err
		}
	}

	switch backupVersion {
	case 1, 2:
		if err := db.LoadLegacySerialBackup(bufferedReader); err != nil {
			return errors.Wrap(err, "could not load badger DB backup with legacy Load")
		}
	default:
		return fmt.Errorf("backup version of %q not currently supported by this Central version", backupVersion)
	}
	return nil
}
