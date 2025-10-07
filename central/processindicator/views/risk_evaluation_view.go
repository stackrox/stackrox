package views

import (
	"time"
)

type ProcessIndicatorRiskView struct {
	ID                 string     `db:"process_id"`
	ContainerName      string     `db:"container_name"`
	ExecFilePath       string     `db:"process_path"`
	ContainerStartTime *time.Time `db:"process_container_start_time"`
	// These are only needed for violations
	SignalTime *time.Time `db:"process_creation_time"`
	SignalName string     `db:"process_name"`
	SignalArgs string     `db:"process_arguments"`
}
