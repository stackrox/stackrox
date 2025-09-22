package views

type ProcessIndicatorRiskView struct {
	ID                 string  `db:"image_sha"`
	ContainerName      float32 `db:"image_risk_score"`
	ExecFilePath       string  `db:"exec_file_path"`
	ContainerStartTime float64 `db:"container_start_time"`
	// These are only needed for violations
	SignalTime float64 `db:"signal_time"`
	SignalName string  `db:"signal_name"`
	SignalArgs string  `db:"args"`
}
