package cache

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stretchr/testify/assert"
)

type getKeyTestCase struct {
	name                         string
	image                        *storage.ContainerImage
	expectedWithoutFlattenImgCap Key
	expectedWithFlattenImgCap    Key
}

func getKeyTestCases() []getKeyTestCase {
	imageName1 := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}
	imageName2 := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "stackrox/main",
		Tag:      "4.0.0",
		FullName: "quay.io/stackrox/main:4.0.0",
	}

	return []getKeyTestCase{
		{
			name: "image with ID returns ID as key without cap, UUID V5 with cap",
			image: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: Key("sha256:abc123"),
			expectedWithFlattenImgCap:    Key(utils.NewImageV2ID(imageName1, "sha256:abc123")),
		},
		{
			name: "image without ID returns full name as key",
			image: &storage.ContainerImage{
				Id:   "",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: Key("docker.io/library/nginx:latest"),
			expectedWithFlattenImgCap:    Key("docker.io/library/nginx:latest"),
		},
		{
			name: "image with ID and imageName2 returns ID as key without cap, UUID V5 with cap",
			image: &storage.ContainerImage{
				Id:   "sha256:def456",
				Name: imageName2,
			},
			expectedWithoutFlattenImgCap: Key("sha256:def456"),
			expectedWithFlattenImgCap:    Key(utils.NewImageV2ID(imageName2, "sha256:def456")),
		},
	}
}

func TestGetKey_WithoutFlattenImageCapability(t *testing.T) {
	// Set up: ensure FlattenImageData capability is not set
	centralcaps.Set([]centralsensor.CentralCapability{})

	for _, tt := range getKeyTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKey(tt.image)
			assert.Equal(t, tt.expectedWithoutFlattenImgCap, result)
		})
	}
}

func TestGetKey_WithFlattenImageCapability(t *testing.T) {
	// Set up: enable FlattenImageData capability
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})

	for _, tt := range getKeyTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKey(tt.image)
			assert.Equal(t, tt.expectedWithFlattenImgCap, result)
		})
	}
}

type compareKeysTestCase struct {
	name                         string
	a                            *storage.ContainerImage
	b                            *storage.ContainerImage
	expectedWithoutFlattenImgCap bool
	expectedWithFlattenImgCap    bool
}

func compareKeysTestCases() []compareKeysTestCase {
	imageName1 := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}
	imageName2 := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "stackrox/main",
		Tag:      "4.0.0",
		FullName: "quay.io/stackrox/main:4.0.0",
	}

	return []compareKeysTestCase{
		{
			name: "same ID and name returns true",
			a: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: true,
			expectedWithFlattenImgCap:    true,
		},
		{
			name: "different IDs returns false",
			a: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "sha256:def456",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: false,
			expectedWithFlattenImgCap:    false,
		},
		{
			name: "different names with same ID returns true without cap, false with cap",
			a: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName2,
			},
			expectedWithoutFlattenImgCap: true,  // Compares only IDs
			expectedWithFlattenImgCap:    false, // Compares UUID V5 (name + ID)
		},
		{
			name: "same full name without ID returns true",
			a: &storage.ContainerImage{
				Id:   "",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: true,
			expectedWithFlattenImgCap:    true,
		},
		{
			name: "different full names without ID returns false",
			a: &storage.ContainerImage{
				Id:   "",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "",
				Name: imageName2,
			},
			expectedWithoutFlattenImgCap: false,
			expectedWithFlattenImgCap:    false,
		},
		{
			name: "one with ID, one without ID with same name returns true (compares names)",
			a: &storage.ContainerImage{
				Id:   "sha256:abc123",
				Name: imageName1,
			},
			b: &storage.ContainerImage{
				Id:   "",
				Name: imageName1,
			},
			expectedWithoutFlattenImgCap: true,
			expectedWithFlattenImgCap:    true,
		},
	}
}

func TestCompareKeys_WithoutFlattenImageCapability(t *testing.T) {
	// Set up: ensure FlattenImageData capability is not set
	centralcaps.Set([]centralsensor.CentralCapability{})

	for _, tt := range compareKeysTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareKeys(tt.a, tt.b)
			assert.Equal(t, tt.expectedWithoutFlattenImgCap, result)
		})
	}
}

func TestCompareKeys_WithFlattenImageCapability(t *testing.T) {
	// Set up: enable FlattenImageData capability
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})

	for _, tt := range compareKeysTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareKeys(tt.a, tt.b)
			assert.Equal(t, tt.expectedWithFlattenImgCap, result)
		})
	}
}

func TestGetKey_CapabilityToggle(t *testing.T) {
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}
	image := &storage.ContainerImage{
		Id:   "sha256:abc123",
		Name: imageName,
	}

	// Test without capability
	centralcaps.Set([]centralsensor.CentralCapability{})
	keyWithoutCap := GetKey(image)
	assert.Equal(t, Key("sha256:abc123"), keyWithoutCap)

	// Test with capability
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})
	keyWithCap := GetKey(image)
	expectedUUID := utils.NewImageV2ID(imageName, "sha256:abc123")
	assert.Equal(t, Key(expectedUUID), keyWithCap)

	// Verify they are different
	assert.NotEqual(t, keyWithoutCap, keyWithCap)
}

func TestCompareKeys_CapabilityToggle(t *testing.T) {
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}
	a := &storage.ContainerImage{
		Id:   "sha256:abc123",
		Name: imageName,
	}
	b := &storage.ContainerImage{
		Id:   "sha256:abc123",
		Name: imageName,
	}

	// Test without capability
	centralcaps.Set([]centralsensor.CentralCapability{})
	assert.True(t, CompareKeys(a, b), "should be equal without capability")

	// Test with capability
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.FlattenImageData})
	assert.True(t, CompareKeys(a, b), "should be equal with capability")
}
