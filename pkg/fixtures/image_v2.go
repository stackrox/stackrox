package fixtures

import (
	"fmt"
	"math/rand"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetImageV2 returns a Mock ImageV2
func GetImageV2() *storage.ImageV2 {
	numComponentsPerImage := 50
	componentsPerImage := make([]*storage.EmbeddedImageScanComponent, 0, numComponentsPerImage)
	for i := 0; i < numComponentsPerImage; i++ {
		eisc := &storage.EmbeddedImageScanComponent{}
		eisc.SetName("name")
		eisc.SetVersion("1.2.3.4")
		eisc.SetVulns(getVulnsPerComponent(i, 5, storage.EmbeddedVulnerability_IMAGE_VULNERABILITY))
		componentsPerImage = append(componentsPerImage, eisc)
	}
	return getImageWithComponentsV2(componentsPerImage)
}

func GetImageV2withDulicateVulnerabilities() *storage.ImageV2 {
	numComponentsPerImage := 50
	componentsPerImage := make([]*storage.EmbeddedImageScanComponent, 0, numComponentsPerImage)
	for i := 0; i < numComponentsPerImage; i++ {
		eisc := &storage.EmbeddedImageScanComponent{}
		eisc.SetName("name")
		eisc.SetVersion("1.2.3.4")
		eisc.SetVulns(getVulnsPerComponent(i, 5, storage.EmbeddedVulnerability_IMAGE_VULNERABILITY))
		componentsPerImage = append(componentsPerImage, eisc)
	}
	cveName := fmt.Sprintf("CVE-Duplicate-2025-%04d", rand.Intn(10_000))
	duplicateVuln := &storage.EmbeddedVulnerability{}
	duplicateVuln.SetCve(cveName)
	duplicateVuln.SetCvss(5)
	duplicateVuln.SetVulnerabilityType(storage.EmbeddedVulnerability_IMAGE_VULNERABILITY)
	duplicateVuln.SetSeverity(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY)
	duplicateVuln.SetSummary("Duplicate CVE for testing")
	duplicateVuln.SetLink(fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveName))
	duplicateVuln.Set_FixedBy("abcdefg")
	componentsPerImage[0].SetVulns(append(componentsPerImage[0].GetVulns(), duplicateVuln))
	componentsPerImage[1].SetVulns(append(componentsPerImage[1].GetVulns(), duplicateVuln))

	return getImageWithComponentsV2(componentsPerImage)
}

// GetImageV2WithUniqueComponents returns a Mock Image where each component is unique
func GetImageV2WithUniqueComponents(numComponents int) *storage.ImageV2 {
	componentsPerImage := make([]*storage.EmbeddedImageScanComponent, 0, numComponents)
	for i := 0; i < numComponents; i++ {
		eisc := &storage.EmbeddedImageScanComponent{}
		eisc.SetName(fmt.Sprintf("name-%d", i))
		eisc.SetVersion(fmt.Sprintf("%d.2.3.4", i))
		eisc.SetVulns(getVulnsPerComponent(i, 5, storage.EmbeddedVulnerability_IMAGE_VULNERABILITY))
		componentsPerImage = append(componentsPerImage, eisc)
	}
	return getImageWithComponentsV2(componentsPerImage)
}

func getImageWithComponentsV2(componentsPerImage []*storage.EmbeddedImageScanComponent) *storage.ImageV2 {
	author := "author"
	imageName := "stackrox.io/srox/mongo:latest"
	imageSha := "sha256:SHA2"
	imageID := uuid.NewV5FromNonUUIDs(imageName, imageSha).String()
	return storage.ImageV2_builder{
		Id:     imageID,
		Digest: imageSha,
		Name: storage.ImageName_builder{
			Registry: "stackrox.io",
			Remote:   "srox/mongo",
			Tag:      "latest",
			FullName: imageName,
		}.Build(),
		Metadata: storage.ImageMetadata_builder{
			V1: storage.V1Metadata_builder{
				Author:  author,
				Created: protocompat.TimestampNow(),
				Layers: []*storage.ImageLayer{
					storage.ImageLayer_builder{
						Instruction: "CMD",
						Value:       `["nginx" "-g" "daemon off;"]`,
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "EXPOSE",
						Value:       "443/tcp 80/tcp",
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "RUN",
						Value:       "ln -sf /dev/stdout /var/log/nginx/access.log && ln -sf /dev/stderr /var/log/nginx/error.log",
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "RUN",
						Value:       `apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62 && echo "deb http://nginx.org/packages/debian/ jessie nginx" >> /etc/apt/sources.list && apt-get update && apt-get install --no-install-recommends --no-install-suggests -y ca-certificates nginx=${NGINX_VERSION} nginx-module-xslt nginx-module-geoip nginx-module-image-filter nginx-module-perl nginx-module-njs gettext-base && rm -rf /var/lib/apt/lists/*`,
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ENV",
						Value:       "NGINX_VERSION=1.10.3-1~jessie",
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "MAINTAINER",
						Value:       author,
						Author:      author,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "CMD",
						Value:       `["/bin/bash"]`,
						Created:     protocompat.TimestampNow(),
					}.Build(),
					storage.ImageLayer_builder{
						Instruction: "ADD",
						Value:       "file:4eedf861fb567fffb2694b65ebdd58d5e371a2c28c3863f363f333cb34e5eb7b in /",
						Created:     protocompat.TimestampNow(),
					}.Build(),
				},
			}.Build(),
			V2: storage.V2Metadata_builder{
				Digest: "sha256:0346349a1a640da9535acfc0f68be9d9b81e85957725ecb76f3b522f4e2f0455",
			}.Build(),
			LayerShas: []string{
				"sha256:6d827a3ef358f4fa21ef8251f95492e667da826653fd43641cef5a877dc03a70",
				"sha256:1e3e18a64ea9924fd9688d125c2844c4df144e41b1d2880a06423bca925b778c",
				"sha256:556c62bb43ac9073f4dfc95383e83f8048633a041cb9e7eb2c1f346ba39a5183",
			},
		}.Build(),
		Scan: storage.ImageScan_builder{
			ScanTime:   protocompat.TimestampNow(),
			Components: componentsPerImage,
		}.Build(),
		ScanStats: storage.ImageV2_ScanStats_builder{
			ComponentCount: int32(len(componentsPerImage)),
		}.Build(),
	}.Build()
}
