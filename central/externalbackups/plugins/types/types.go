package types

import (
	"io"
)

// ExternalBackup defines the interface that all external backups must implement
type ExternalBackup interface {
	Backup(reader io.ReadCloser) error
	Restore() error
	Test() error
}
