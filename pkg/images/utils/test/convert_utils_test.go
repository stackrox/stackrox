package test

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConvertUtils(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	suite.Run(t, new(TestConvertUtilsSuite))
}

type TestConvertUtilsSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *TestConvertUtilsSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *TestConvertUtilsSuite) TestConvertV2NotesToV1Notes() {
	inputs := []storage.ImageV2_Note{
		storage.ImageV2_MISSING_METADATA,
		storage.ImageV2_MISSING_SCAN_DATA,
		storage.ImageV2_MISSING_SIGNATURE,
		storage.ImageV2_MISSING_SIGNATURE_VERIFICATION_DATA,
	}
	expected := []storage.Image_Note{
		storage.Image_MISSING_METADATA,
		storage.Image_MISSING_SCAN_DATA,
		storage.Image_MISSING_SIGNATURE,
		storage.Image_MISSING_SIGNATURE_VERIFICATION_DATA,
	}
	outputs := utils.ConvertNotesToV1(inputs)
	s.Equal(expected, outputs)
}

func (s *TestConvertUtilsSuite) TestConvertV2ImageToV1() {
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/alpine",
		Tag:      "latest",
		FullName: "docker.io/library/alpine:latest",
	}
	timestamp := timestamppb.Now()
	image := &storage.ImageV2{
		Id:   fixtureconsts.Deployment1,
		Sha:  "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		Name: imageName,
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Digest:  "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Created: timestamp,
				Author:  "StackRox",
				Layers: []*storage.ImageLayer{
					{
						Instruction: "FROM",
						Value:       "alpine:latest",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					},
					{
						Instruction: "ENTRYPOINT",
						Value:       "/bin/sh",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					},
				},
				User:       "stackrox",
				Command:    []string{"/bin/sh"},
				Entrypoint: []string{"/bin/sh"},
				Volumes:    []string{},
				Labels: map[string]string{
					"maintainer": "RedHat Advanced Cluster Security",
				},
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			},
			LayerShas: []string{
				"sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				"sha256:30b7ec5d0cffb9b3ce785c0574473a79d92ea75a6e1e8ff1d0c2372919f9112d",
			},
			DataSource: &storage.DataSource{
				Id:     "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
				Name:   "Public DockerHub",
				Mirror: "",
			},
			Version: 0,
		},
		Scan:                      &storage.ImageScan{},
		SignatureVerificationData: nil,
		Signature: &storage.ImageSignature{
			Signatures: nil,
			Fetched:    nil,
		},
		ComponentCount:  150,
		CveCount:        175,
		FixableCveCount: 200,
		LastUpdated:     timestamp,
		NotPullable:     false,
		TopCvss:         9.5,
		RiskScore:       10.5,
		Notes: []storage.ImageV2_Note{
			storage.ImageV2_MISSING_SIGNATURE,
		},
	}
	expected := &storage.Image{
		Id:   "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		Name: imageName,
		Names: []*storage.ImageName{
			imageName,
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Digest:  "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Created: timestamp,
				Author:  "StackRox",
				Layers: []*storage.ImageLayer{
					{
						Instruction: "FROM",
						Value:       "alpine:latest",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					},
					{
						Instruction: "ENTRYPOINT",
						Value:       "/bin/sh",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					},
				},
				User:       "stackrox",
				Command:    []string{"/bin/sh"},
				Entrypoint: []string{"/bin/sh"},
				Volumes:    []string{},
				Labels: map[string]string{
					"maintainer": "RedHat Advanced Cluster Security",
				},
			},
			V2: &storage.V2Metadata{
				Digest: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			},
			LayerShas: []string{
				"sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				"sha256:30b7ec5d0cffb9b3ce785c0574473a79d92ea75a6e1e8ff1d0c2372919f9112d",
			},
			DataSource: &storage.DataSource{
				Id:     "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
				Name:   "Public DockerHub",
				Mirror: "",
			},
			Version: 0,
		},
		Scan:                      &storage.ImageScan{},
		SignatureVerificationData: nil,
		Signature: &storage.ImageSignature{
			Signatures: nil,
			Fetched:    nil,
		},
		SetComponents: &storage.Image_Components{
			Components: 150,
		},
		SetCves: &storage.Image_Cves{
			Cves: 175,
		},
		SetFixable: &storage.Image_FixableCves{
			FixableCves: 200,
		},
		LastUpdated: timestamp,
		NotPullable: false,
		SetTopCvss: &storage.Image_TopCvss{
			TopCvss: 9.5,
		},
		RiskScore: 10.5,
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE,
		},
	}
	result := utils.ConvertToV1(image)
	protoassert.Equal(s.T(), expected, result)
}
