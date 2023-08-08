package pruning

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDownloadableReportPruning(t *testing.T) {
	config := &storage.PrivateConfig{
		ReportRetentionConfig: &storage.ReportRetentionConfig{
			DownloadableReportRetentionDays:        7,
			DownloadableReportGlobalRetentionBytes: 500,
		},
	}
	ctrl := gomock.NewController(t)
	mockBlobStore := blobDSMocks.NewMockDatastore(ctrl)
	gci := &garbageCollectorImpl{
		blobStore: mockBlobStore,
	}

	type blobTemplate struct {
		name             string
		modTimeMinusDays int
		length           int64
	}

	expiredBlob := &blobTemplate{
		name:             "/central/reports/exp/8",
		modTimeMinusDays: 8,
		length:           0,
	}
	bigBlob := &blobTemplate{
		name:             "/central/reports/big/3",
		modTimeMinusDays: 3,
		length:           501,
	}
	mediumBlobs := []*blobTemplate{
		{
			name:             "/central/reports/m/0",
			modTimeMinusDays: 0,
			length:           200,
		},
		{
			name:             "/central/reports/m/1",
			modTimeMinusDays: 1,
			length:           200,
		},
		{
			name:             "/central/reports/m/2",
			modTimeMinusDays: 2,
			length:           200,
		},
		{
			name:             "/central/reports/m/4",
			modTimeMinusDays: 4,
			length:           200,
		},
		{
			name:             "/central/reports/m/5",
			modTimeMinusDays: 5,
			length:           200,
		},
		{
			name:             "/central/reports/m/6",
			modTimeMinusDays: 6,
			length:           200,
		},
	}

	testCases := []struct {
		description    string
		existing       []*blobTemplate
		toRemove       []*blobTemplate
		retentionBytes uint32
	}{
		{
			description: "nothing to prune",
		},
		{
			description: "prune big blob",
			existing: []*blobTemplate{
				mediumBlobs[0],
				bigBlob,
				mediumBlobs[5],
			},
			toRemove: []*blobTemplate{bigBlob, mediumBlobs[5]},
		},
		{
			description: "prune expired blob",
			existing: []*blobTemplate{
				mediumBlobs[0],
				expiredBlob,
				mediumBlobs[1],
			},
			toRemove: []*blobTemplate{expiredBlob},
		},
		{
			description: "prune for accumulated size",
			existing: append(mediumBlobs, []*blobTemplate{
				expiredBlob,
			}...),
			toRemove: append(mediumBlobs[2:], expiredBlob),
		},
		{
			description: "prune for accumulated size keep most",
			existing: append(mediumBlobs, []*blobTemplate{
				expiredBlob,
				bigBlob,
			}...),
			toRemove:       append(mediumBlobs[4:], expiredBlob),
			retentionBytes: 1500,
		},
	}

	now := time.Now()
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			currConfig := config.Clone()
			if c.retentionBytes != 0 {
				currConfig.ReportRetentionConfig.DownloadableReportGlobalRetentionBytes = c.retentionBytes
			}
			var existingBlobs []*storage.Blob
			for _, bt := range c.existing {
				modTime, err := types.TimestampProto(now.Add(-time.Duration(bt.modTimeMinusDays) * 24 * time.Hour))
				require.NoError(t, err)
				existingBlobs = append(existingBlobs, &storage.Blob{
					Name:         bt.name,
					Length:       bt.length,
					ModifiedTime: modTime,
				})
			}
			rand.Shuffle(len(existingBlobs), func(i, j int) {
				temp := existingBlobs[i]
				existingBlobs[i] = existingBlobs[j]
				existingBlobs[j] = temp
			})
			toRemoveSet := set.NewStringSet()
			for _, bt := range c.toRemove {
				toRemoveSet.Add(bt.name)
			}
			mockBlobStore.EXPECT().SearchMetadata(gomock.Any(), gomock.Any()).Times(1).Return(existingBlobs, nil)
			mockBlobStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(toRemoveSet.Cardinality()).DoAndReturn(func(ctx context.Context, name string) error {
				assert.True(t, toRemoveSet.Remove(name), "unexpected blob %s deleted", name)
				return nil
			})
			gci.removeOldReportBlobs(currConfig)
			assert.Zero(t, toRemoveSet.Cardinality(), "blob should be deleted %v", toRemoveSet)
		})
	}
}
