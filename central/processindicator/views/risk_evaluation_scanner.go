package views

import (
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// RiskViewSelectProtos defines the column projections for process indicator
// risk view queries. Order must match the scan destinations in
// ProcessIndicatorRiskScanner.Dests.
var RiskViewSelectProtos = []*v1.QuerySelect{
	search.NewQuerySelect(search.ProcessID).Proto(),
	search.NewQuerySelect(search.ContainerName).Proto(),
	search.NewQuerySelect(search.ProcessExecPath).Proto(),
	search.NewQuerySelect(search.ProcessContainerStartTime).Proto(),
	search.NewQuerySelect(search.ProcessCreationTime).Proto(),
	search.NewQuerySelect(search.ProcessName).Proto(),
	search.NewQuerySelect(search.ProcessArguments).Proto(),
}

// ProcessIndicatorRiskScanner holds pgx scan destinations for direct row
// scanning, bypassing scany reflection.
type ProcessIndicatorRiskScanner struct {
	ID                 string
	ContainerName      string
	ExecFilePath       string
	ContainerStartTime pgtype.Timestamp
	SignalTime         pgtype.Timestamp
	SignalName         string
	SignalArgs         string
}

// Dests returns scan destination pointers in the order matching
// RiskViewSelectProtos.
func (s *ProcessIndicatorRiskScanner) Dests() []any {
	return []any{
		&s.ID, &s.ContainerName, &s.ExecFilePath,
		&s.ContainerStartTime, &s.SignalTime,
		&s.SignalName, &s.SignalArgs,
	}
}

// Build converts scanned values into a ProcessIndicatorRiskView.
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
