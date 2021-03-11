package check

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func Test_reportCheckResults(t *testing.T) {
	type args struct {
		json                   bool
		failViolationsWithJSON bool
		alerts                 []*storage.Alert
	}
	noFatalAlerts := []*storage.Alert{
		{
			Id: "1234",
			Policy: &storage.Policy{
				Name: "some-policy",
			},
		},
	}
	fatalAlerts := []*storage.Alert{
		{
			Id: "1234",
			Policy: &storage.Policy{
				Name: "some-policy",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
				},
			},
		},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"legacy-nonJSON-noFatalAlerts", args{false, false, noFatalAlerts}, false},
		{"legacy-nonJSON-fatalAlerts", args{false, false, fatalAlerts}, true},
		{"legacy-JSON-noFatalAlerts", args{true, false, noFatalAlerts}, false},
		{"legacy-JSON-fatalAlerts", args{true, false, fatalAlerts}, false},
		{"fixed-nonJSON-noFatalAlerts", args{false, true, noFatalAlerts}, false},
		{"fixed-nonJSON-fatalAlerts", args{false, true, fatalAlerts}, true},
		{"fixed-JSON-noFatalAlerts", args{true, true, noFatalAlerts}, false},
		{"fixed-JSON-fatalAlerts", args{true, true, fatalAlerts}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := reportCheckResults("fake-image", tt.args.json, tt.args.failViolationsWithJSON, tt.args.alerts, false); (err != nil) != tt.wantErr {
				t.Errorf("reportCheckResults() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
