package hash

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/random"
)

func TestHasher(t *testing.T) {
	hasher := NewHasher()

	dep := &storage.Deployment{
		Containers: []*storage.Container{
			{
				Image: &storage.ContainerImage{
					Name: &storage.ImageName{
						Tag: "abc",
					},
				},
			},
		},
	}

	evt := &central.SensorEvent{
		Id:     "",
		Action: 0,
		Resource: &central.SensorEvent_Deployment{
			Deployment: dep,
		},
	}
	val, ok := hasher.HashEvent(evt)
	if !ok {
		panic(ok)
	}
	fmt.Println(val)

	dep.Containers[0].Image.Name.Tag = "def"
	val, ok = hasher.HashEvent(evt)
	if !ok {
		panic(ok)
	}

	prevTag := "def"
	prevHash := val
	for i := 0; i < 10000000; i++ {
		var tag string
		var err error
		for true {
			tag, err = random.GenerateString(3, random.AlphanumericCharacters)
			if err != nil {
				panic(err)
			}
			if tag != prevTag {
				break
			}
		}
		dep.Containers[0].Image.Name.Tag = tag
		val, ok = hasher.HashEvent(evt)
		if !ok {
			panic(ok)
		}
		if prevHash == val {
			fmt.Println("Same hash", prevTag, tag)
			panic("no")
		}
	}
}

/*
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

*/
