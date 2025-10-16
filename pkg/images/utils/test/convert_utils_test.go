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
	"google.golang.org/protobuf/proto"
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
	imageName := &storage.ImageName{}
	imageName.SetRegistry("docker.io")
	imageName.SetRemote("library/alpine")
	imageName.SetTag("latest")
	imageName.SetFullName("docker.io/library/alpine:latest")
	timestamp := timestamppb.Now()
	image := storage.ImageV2_builder{
		Id:     fixtureconsts.Deployment1,
		Digest: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		Name:   imageName,
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Created: timestamp,
				Author:  "StackRox",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "FROM",
						Value:       "alpine:latest",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ENTRYPOINT",
						Value:       "/bin/sh",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					}.Build(),
				},
				User:       "stackrox",
				Command:    []string{"/bin/sh"},
				Entrypoint: []string{"/bin/sh"},
				Volumes:    []string{},
				Labels: map[string]string{
					"maintainer": "RedHat Advanced Cluster Security",
				},
			}.Build(),
			V2: storage.V2Metadata_builder{
				Digest: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			}.Build(),
			LayerShas: []string{
				"sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				"sha256:30b7ec5d0cffb9b3ce785c0574473a79d92ea75a6e1e8ff1d0c2372919f9112d",
			},
			DataSource: storage.DataSource_builder{
				Id:     "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
				Name:   "Public DockerHub",
				Mirror: "",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan:                      &storage.ImageScan{},
		SignatureVerificationData: nil,
		Signature: storage.ImageSignature_builder{
			Signatures: nil,
			Fetched:    nil,
		}.Build(),
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount:  150,
			CveCount:        175,
			FixableCveCount: 200,
		}.Build(),
		LastUpdated: timestamp,
		NotPullable: false,
		TopCvss:     9.5,
		RiskScore:   10.5,
		Notes: []storage.ImageV2_Note{
			storage.ImageV2_MISSING_SIGNATURE,
		},
	}.Build()
	expected := storage.Image_builder{
		Id:   "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		Name: imageName,
		Names: []*storage.ImageName{
			imageName,
		},
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Digest:  "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				Created: timestamp,
				Author:  "StackRox",
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "FROM",
						Value:       "alpine:latest",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ENTRYPOINT",
						Value:       "/bin/sh",
						Created:     timestamp,
						Author:      "StackRox",
						Empty:       true,
					}.Build(),
				},
				User:       "stackrox",
				Command:    []string{"/bin/sh"},
				Entrypoint: []string{"/bin/sh"},
				Volumes:    []string{},
				Labels: map[string]string{
					"maintainer": "RedHat Advanced Cluster Security",
				},
			}.Build(),
			V2: storage.V2Metadata_builder{
				Digest: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			}.Build(),
			LayerShas: []string{
				"sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
				"sha256:30b7ec5d0cffb9b3ce785c0574473a79d92ea75a6e1e8ff1d0c2372919f9112d",
			},
			DataSource: storage.DataSource_builder{
				Id:     "10d3b4dc-8295-41bc-bb50-6da5484cdb1a",
				Name:   "Public DockerHub",
				Mirror: "",
			}.Build(),
			Version: 0,
		}.Build(),
		Scan:                      &storage.ImageScan{},
		SignatureVerificationData: nil,
		Signature: storage.ImageSignature_builder{
			Signatures: nil,
			Fetched:    nil,
		}.Build(),
		Components:  proto.Int32(150),
		Cves:        proto.Int32(175),
		FixableCves: proto.Int32(200),
		LastUpdated: timestamp,
		NotPullable: false,
		TopCvss:     proto.Float32(9.5),
		RiskScore:   10.5,
		Notes: []storage.Image_Note{
			storage.Image_MISSING_SIGNATURE,
		},
	}.Build()
	result := utils.ConvertToV1(image)
	protoassert.Equal(s.T(), expected, result)
}
