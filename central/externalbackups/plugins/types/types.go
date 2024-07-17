package types

import (
	"io"
)

const (
	// S3Type represents the AWS S3 backup typ.
	S3Type = "s3"

	// S3CompatibleType represents the S3 compatible backup typ.
	S3CompatibleType = "s3compatible"

	// GCSType represents the Google cloud storage backup typ.
	GCSType = "gcs"
)

// ExternalBackup defines the interface that all external backups must implement
type ExternalBackup interface {
	Backup(reader io.ReadCloser) error
	Test() error
}
