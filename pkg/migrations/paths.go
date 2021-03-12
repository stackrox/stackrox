package migrations

const (
	// DBMountPath is the directory path (within a container) where database storage device is mounted.
	DBMountPath = "/var/lib/stackrox"

	// CurrentPath is the link (within a container) to current migration directory. This directory contains
	// databases and other migration related contents.
	CurrentPath = DBMountPath + "/current"
)
