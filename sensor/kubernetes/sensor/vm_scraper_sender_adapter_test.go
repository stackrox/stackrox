package sensor

import (
	"context"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVMScraperSenderAdapter_Send(t *testing.T) {
	cid := uint32(100)

	cases := map[string]struct {
		vm           *virtualmachine.Info
		wantErr      bool
		wantVmID     string
		wantVsockCid string
	}{
		"should set vm_id when CID is nil": {
			vm: &virtualmachine.Info{
				ID:       "vm-1",
				VSOCKCID: nil,
			},
			wantVmID:     "vm-1",
			wantVsockCid: "",
		},
		"should include CID when known": {
			vm: &virtualmachine.Info{
				ID:       "vm-1",
				VSOCKCID: &cid,
			},
			wantVmID:     "vm-1",
			wantVsockCid: "100",
		},
		"should return error when VM is nil": {
			vm:      nil,
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			handler := mocks.NewMockHandler(ctrl)
			adapter := &vmScraperSenderAdapter{handler: handler}

			if !tc.wantErr {
				handler.EXPECT().
					Send(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, report *v1.IndexReport, _ time.Time) error {
						assert.Equal(t, tc.wantVmID, report.GetVmId())
						assert.Equal(t, tc.wantVsockCid, report.GetVsockCid())
						assert.Equal(t, "IndexFinished", report.GetIndexV4().GetState())
						return nil
					})
			}

			err := adapter.Send(t.Context(), tc.vm, &v4.IndexReport{State: "IndexFinished"}, time.Time{})
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
