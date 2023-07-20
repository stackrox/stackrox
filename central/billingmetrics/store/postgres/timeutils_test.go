package postgres

import (
	"testing"
	"time"
)

func TestTimeUtils(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)

	toStore := timeToTimestamp(now)
	stored := timestampToStoreInUTC(toStore)
	retrieved := localTimestampFromStore(&stored)

	if toStore.Compare(retrieved) != 0 {
		t.Error(toStore, retrieved)
	}
}
