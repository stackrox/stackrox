package datastore

import (
	"testing"
	"time"

	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

func TestGetSuppressExpiry(t *testing.T) {
	startTime := time.Now().UTC()
	duration := 10 * time.Minute

	expiry1, err := getSuppressExpiry(nil, nil)
	assert.ErrorIs(t, err, errNilSuppressionStart)
	assert.Nil(t, expiry1)

	expiry2, err := getSuppressExpiry(nil, &duration)
	assert.ErrorIs(t, err, errNilSuppressionStart)
	assert.Nil(t, expiry2)

	expiry3, err := getSuppressExpiry(&startTime, nil)
	assert.ErrorIs(t, err, errNilSuppressionDuration)
	assert.Nil(t, expiry3)

	expiry4, err := getSuppressExpiry(&startTime, &duration)
	assert.NoError(t, err)
	truncatedStart := startTime.Truncate(time.Second)
	truncatedDuration := duration.Truncate(time.Second)
	expectedExpiry4 := truncatedStart.Add(truncatedDuration)
	assert.Equal(t, &expectedExpiry4, expiry4)
}

func TestGetSuppressionCacheEntry(t *testing.T) {
	startTime := time.Now().UTC()
	duration := 10 * time.Minute
	activation := startTime.Truncate(time.Nanosecond)
	expiration := startTime.Add(duration)

	protoStart, err := protocompat.ConvertTimeToTimestampOrError(startTime)
	assert.NoError(t, err)
	protoExpiration, err := protocompat.ConvertTimeToTimestampOrError(expiration)
	assert.NoError(t, err)

	cve1 := &storage.ClusterCVE{}
	expectedEntry1 := common.SuppressionCacheEntry{}
	entry1 := getSuppressionCacheEntry(cve1)
	assert.Equal(t, expectedEntry1, entry1)

	cve2 := &storage.ClusterCVE{
		SnoozeStart: protoStart,
	}
	expectedEntry2 := common.SuppressionCacheEntry{
		SuppressActivation: &activation,
	}
	entry2 := getSuppressionCacheEntry(cve2)
	assert.Equal(t, expectedEntry2, entry2)

	cve3 := &storage.ClusterCVE{
		SnoozeExpiry: protoExpiration,
	}
	expectedEntry3 := common.SuppressionCacheEntry{
		SuppressExpiry: &expiration,
	}
	entry3 := getSuppressionCacheEntry(cve3)
	assert.Equal(t, expectedEntry3, entry3)

	cve4 := &storage.ClusterCVE{
		SnoozeStart:  protoStart,
		SnoozeExpiry: protoExpiration,
	}
	expectedEntry4 := common.SuppressionCacheEntry{
		SuppressActivation: &activation,
		SuppressExpiry:     &expiration,
	}
	entry4 := getSuppressionCacheEntry(cve4)
	assert.Equal(t, expectedEntry4, entry4)
}
