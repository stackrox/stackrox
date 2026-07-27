package cmd

import (
	"testing"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/common"
	"github.com/stretchr/testify/assert"
)

func TestValidateDaemonConfig(t *testing.T) {
	tests := map[string]struct {
		cfg     *common.Config
		wantErr bool
	}{
		"should pass when daemon mode is disabled regardless of index interval": {
			cfg:     &common.Config{DaemonMode: false, IndexInterval: time.Second},
			wantErr: false,
		},
		"should pass when daemon mode index interval meets the minimum": {
			cfg:     &common.Config{DaemonMode: true, IndexInterval: minDaemonIndexInterval},
			wantErr: false,
		},
		"should pass when daemon mode index interval exceeds the minimum": {
			cfg:     &common.Config{DaemonMode: true, IndexInterval: minDaemonIndexInterval * 2},
			wantErr: false,
		},
		"should error when daemon mode index interval is below the minimum": {
			cfg:     &common.Config{DaemonMode: true, IndexInterval: minDaemonIndexInterval - time.Second},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateDaemonConfig(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
