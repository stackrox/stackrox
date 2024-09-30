package scannerv4

import (
	"errors"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/protocompat"
	s4ClientMocks "github.com/stackrox/rox/pkg/scannerv4/client/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetVulnDefinitionsInfo(t *testing.T) {
	errExpected := true
	var noErr error
	var noMetadata *v4.Metadata
	tmsFromTime, _ := protocompat.ConvertTimeToTimestampOrError(time.Time{})
	testCases := []struct {
		desc         string
		clientRet    *v4.Metadata
		clientRetErr error
		errExpected  bool
	}{
		{
			"error when client returns error",
			noMetadata, errors.New("fake"), errExpected,
		},
		{
			"error when client returns nil metadata",
			noMetadata, noErr, errExpected,
		},
		{
			"error when client returns nil last vuln update",
			&v4.Metadata{}, noErr, errExpected,
		},
		{
			"error when client returns zero last vuln update",
			&v4.Metadata{LastVulnerabilityUpdate: protocompat.GetProtoTimestampZero()}, noErr, errExpected,
		},
		{
			"error when client returns zero last vuln update (from time)",
			&v4.Metadata{LastVulnerabilityUpdate: tmsFromTime}, noErr, errExpected,
		},
		{
			"success when client returns valid last vuln update",
			&v4.Metadata{LastVulnerabilityUpdate: protocompat.TimestampNow()}, noErr, !errExpected,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			scannerClient := s4ClientMocks.NewMockScanner(ctrl)
			scannerClient.EXPECT().GetMatcherMetadata(gomock.Any()).Return(tc.clientRet, tc.clientRetErr)
			s := scannerv4{scannerClient: scannerClient}

			vdi, err := s.GetVulnDefinitionsInfo()
			if tc.errExpected {
				if tc.clientRetErr != nil {
					require.ErrorContains(t, err, tc.clientRetErr.Error())
				} else {
					require.ErrorContains(t, err, "timestamp")
				}
				assert.Nil(t, vdi)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.clientRet.GetLastVulnerabilityUpdate().AsTime(), vdi.GetLastUpdatedTimestamp().AsTime())
			}
		})
	}

}
