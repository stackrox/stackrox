package hash

import (
	"hash"
	"hash/fnv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/utils"
)

// Hasher provides a unified interface to hashing central.MsgFromSensor
type Hasher struct {
	hasher hash.Hash64
}

// NewHasher creates a new Sensor hasher
func NewHasher() *Hasher {
	return &Hasher{
		hasher: fnv.New64a(),
	}
}

// HashEvent hashes the message from Sensor
func (h *Hasher) HashEvent(event *central.SensorEvent) (uint64, bool) {
	if event == nil {
		return 0, false
	}
	h.hasher.Reset()
	hashValue, err := hashstructure.Hash(event.GetResource(), hashstructure.FormatV2, &hashstructure.HashOptions{
		TagName: "sensorhash",
		Hasher:  h.hasher,
	})
	if err != nil {
		utils.Should(err)
		return 0, false
	}
	return hashValue, true
}
