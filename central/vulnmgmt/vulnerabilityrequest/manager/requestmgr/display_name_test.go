package requestmgr

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestName(t *testing.T) {
	t.Parallel()

	layout := "2006/01/02"
	aug29, err := time.Parse(layout, "2023/08/29")
	require.NoError(t, err)
	aug30, err := time.Parse(layout, "2023/08/30")
	require.NoError(t, err)
	sept29, err := time.Parse(layout, "2023/09/29")
	require.NoError(t, err)

	for _, tc := range []struct {
		desc            string
		req             *storage.VulnerabilityRequest
		lastKnownSeqNum *monthSeqNumPair
		expected        string
	}{
		{
			desc:     "first ever",
			req:      testRequestNameFakeVulnReq(aug29, "bruce wayne"),
			expected: "BW-230829-1",
		},
		{
			desc: "same month",
			req:  testRequestNameFakeVulnReq(aug30, "bruce wayne"),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: aug29.Month(),
				seqNum:           5,
			},
			expected: "BW-230830-6",
		},
		{
			desc: "different month",
			req:  testRequestNameFakeVulnReq(sept29, "bruce wayne"),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: aug29.Month(),
				seqNum:           5,
			},
			expected: "BW-230929-1",
		},
		{
			desc: "one word user name",
			req:  testRequestNameFakeVulnReq(sept29, "bruce"),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: aug29.Month(),
				seqNum:           5,
			},
			expected: "BB-230929-1",
		},
		{
			desc: "multiple words user name",
			req:  testRequestNameFakeVulnReq(sept29, "bruce thomas wayne"),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: aug29.Month(),
				seqNum:           5,
			},
			expected: "BW-230929-1",
		},
		{
			desc: "empty user name",
			req:  testRequestNameFakeVulnReq(sept29, ""),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: aug29.Month(),
				seqNum:           5,
			},
			expected: "SYS-230929-1",
		},
		{
			desc: "whitespaces",
			req:  testRequestNameFakeVulnReq(sept29, "bruce   thomas wayne  "),
			lastKnownSeqNum: &monthSeqNumPair{
				lastCreatedMonth: sept29.Month(),
				seqNum:           5,
			},
			expected: "BW-230929-6",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			actual := requestName(tc.req, tc.lastKnownSeqNum)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func testRequestNameFakeVulnReq(createdAt time.Time, userName string) *storage.VulnerabilityRequest {
	return &storage.VulnerabilityRequest{
		Id: uuid.NewV4().String(),
		Requestor: &storage.SlimUser{
			Name: userName,
		},
		CreatedAt: protoconv.ConvertTimeToTimestamp(createdAt.UTC()),
	}
}
