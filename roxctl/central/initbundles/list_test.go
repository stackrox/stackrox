package initbundles

import (
	"bytes"
	"io"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func Test_outputBundles(t *testing.T) {
	secondsCreatedAt := int64(2222222222)
	secondsExpiresAt := int64(2345678901)
	nanos := int32(123456789)

	tests := map[string]struct {
		bundles []*v1.InitBundleMeta
		output  string
	}{
		"empty list": {
			bundles: []*v1.InitBundleMeta{},
			output: `Name	Created at	Expires at	Created By	ID
====	==========	==========	==========	==
`,
		},
		"time format is good": {
			bundles: []*v1.InitBundleMeta{
				{
					Id:        "9887ccc2-20ce-44f9-b316-57f9f79f5e05",
					Name:      "test",
					CreatedAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsCreatedAt, nanos),
					ExpiresAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsExpiresAt, nanos),
				},
			},
			output: `Name	Created at			Expires at			Created By	ID
====	==========			==========			==========	==
test	2040-06-02T03:57:02.123456789Z	2044-05-01T01:28:21.123456789Z	(unknown)	9887ccc2-20ce-44f9-b316-57f9f79f5e05
`,
		},
		"time format is good without nanos": {
			bundles: []*v1.InitBundleMeta{
				{
					Id:        "9887ccc2-20ce-44f9-b316-57f9f79f5e05",
					Name:      "test",
					CreatedAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsCreatedAt, 0),
					ExpiresAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsExpiresAt, 0),
				},
			},
			output: `Name	Created at		Expires at		Created By	ID
====	==========		==========		==========	==
test	2040-06-02T03:57:02Z	2044-05-01T01:28:21Z	(unknown)	9887ccc2-20ce-44f9-b316-57f9f79f5e05
`,
		},
		"nil Expires at": {
			bundles: []*v1.InitBundleMeta{
				{
					Id:        "9887ccc2-20ce-44f9-b316-57f9f79f5e05",
					Name:      "test",
					CreatedAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsCreatedAt, nanos),
				},
			},
			output: `Name	Created at			Expires at	Created By	ID
====	==========			==========	==========	==
test	2040-06-02T03:57:02.123456789Z	N/A		(unknown)	9887ccc2-20ce-44f9-b316-57f9f79f5e05
`,
		},
		"nil Created at": {
			bundles: []*v1.InitBundleMeta{
				{
					Id:        "9887ccc2-20ce-44f9-b316-57f9f79f5e05",
					Name:      "test",
					ExpiresAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsExpiresAt, nanos),
				},
			},
			output: `Name	Created at	Expires at			Created By	ID
====	==========	==========			==========	==
test	N/A		2044-05-01T01:28:21.123456789Z	(unknown)	9887ccc2-20ce-44f9-b316-57f9f79f5e05
`,
		},
		"full row": {
			bundles: []*v1.InitBundleMeta{
				{
					Id:        "9887ccc2-20ce-44f9-b316-57f9f79f5e05",
					Name:      "test",
					CreatedAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsCreatedAt, nanos),
					CreatedBy: &storage.User{
						Id: "test-user",
					},
					ExpiresAt: protocompat.GetProtoTimestampFromSecondsAndNanos(secondsExpiresAt, 0),
				},
			},
			output: `Name	Created at			Expires at		Created By	ID
====	==========			==========		==========	==
test	2040-06-02T03:57:02.123456789Z	2044-05-01T01:28:21Z	test-user	9887ccc2-20ce-44f9-b316-57f9f79f5e05
`,
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			var b bytes.Buffer
			writer := io.Writer(&b)

			assert.NoError(t, outputBundles(writer, tt.bundles))
			assert.Equal(t, tt.output, b.String())
		})
	}
}
