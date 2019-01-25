package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidityOfRegistry(t *testing.T) {
	a := assert.New(t)

	for startingSeqNum, m := range migrationRegistry {
		a.Equal(startingSeqNum, m.StartingSeqNum)
		a.Equal(startingSeqNum+1, int(m.VersionAfter.GetSeqNum()))
	}
}
