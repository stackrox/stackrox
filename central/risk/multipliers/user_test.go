package multipliers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestUserScore(t *testing.T) {
	mult := NewUserDefined(&storage.Multiplier{
		Scope: &storage.Scope{
			Cluster: "cluster",
		},
		Value: 1.3,
	})
	deployment := getMockDeployment()
	result := mult.Score(deployment)
	assert.Equal(t, float32(1.3), result.GetScore())
	assert.Len(t, result.GetFactors(), 1)

	mult = NewUserDefined(&storage.Multiplier{
		Scope: &storage.Scope{
			Cluster: "blah",
		},
		Value: 1.3,
	})
	result = mult.Score(deployment)
	assert.Nil(t, result)
}
