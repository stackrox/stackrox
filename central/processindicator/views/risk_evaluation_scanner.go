package views

import (
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// RiskViewSelectProtos order must match ProcessIndicatorRiskScanner.Dests.
// See also: central/alert/views/list_alert_scanner.go
var RiskViewSelectProtos = []*v1.QuerySelect{
	search.NewQuerySelect(search.ProcessID).Proto(),
	search.NewQuerySelect(search.ContainerName).Proto(),
	search.NewQuerySelect(search.ProcessExecPath).Proto(),
	search.NewQuerySelect(search.ProcessContainerStartTime).Proto(),
	search.NewQuerySelect(search.ProcessCreationTime).Proto(),
	search.NewQuerySelect(search.ProcessName).Proto(),
	search.NewQuerySelect(search.ProcessArguments).Proto(),
}

// ProcessIndicatorRiskScanner bypasses scany reflection for direct pgx scanning.
type ProcessIndicatorRiskScanner struct {
	ID                 string
	ContainerName      string
	ExecFilePath       string
	ContainerStartTime pgtype.Timestamp
	SignalTime         pgtype.Timestamp
	SignalName         string
	SignalArgs         string
}

func (s *ProcessIndicatorRiskScanner) Dests() []any {
	return []any{
		&s.ID, &s.ContainerName, &s.ExecFilePath,
		&s.ContainerStartTime, &s.SignalTime,
		&s.SignalName, &s.SignalArgs,
	}
}

func (s *ProcessIndicatorRiskScanner) Build() ProcessIndicatorRiskView {
	v := ProcessIndicatorRiskView{
		ID:            s.ID,
		ContainerName: s.ContainerName,
		ExecFilePath:  s.ExecFilePath,
		SignalName:    s.SignalName,
		SignalArgs:    s.SignalArgs,
	}
	if s.ContainerStartTime.Valid {
		t := s.ContainerStartTime.Time.UTC()
		v.ContainerStartTime = &t
	}
	if s.SignalTime.Valid {
		t := s.SignalTime.Time.UTC()
		v.SignalTime = &t
	}
	return v
}
