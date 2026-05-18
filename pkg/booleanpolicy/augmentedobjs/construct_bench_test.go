package augmentedobjs

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func getTestImage(numComponents int) *storage.Image {
	components := make([]*storage.EmbeddedImageScanComponent, 0, numComponents)
	for i := 0; i < numComponents; i++ {
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    fmt.Sprintf("component-%d", i),
			Version: fmt.Sprintf("1.%d.0", i),
			Vulns: []*storage.EmbeddedVulnerability{
				{
					Cve:      fmt.Sprintf("CVE-2024-%04d", i),
					Cvss:     5.0,
					Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				},
			},
		})
	}

	return &storage.Image{
		Id: "sha256:test123",
		Name: &storage.ImageName{
			Registry: "stackrox.io",
			Remote:   "srox/test",
			Tag:      "latest",
			FullName: "stackrox.io/srox/test:latest",
		},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Layers: []*storage.ImageLayer{
					{Instruction: "FROM", Value: "ubuntu:20.04"},
					{Instruction: "RUN", Value: "apt-get update"},
					{Instruction: "RUN", Value: "apt-get install -y nginx"},
					{Instruction: "COPY", Value: ". /app"},
					{Instruction: "WORKDIR", Value: "/app"},
					{Instruction: "EXPOSE", Value: "8080"},
					{Instruction: "CMD", Value: "[\"nginx\", \"-g\", \"daemon off;\"]"},
				},
			},
		},
		Scan: &storage.ImageScan{
			Components: components,
		},
	}
}

func BenchmarkConstructImage(b *testing.B) {
	b.ReportAllocs()
	image := getTestImage(50)
	imageFullName := "stackrox.io/srox/test:latest"

	for b.Loop() {
		_, _ = ConstructImage(image, imageFullName)
	}
}

func BenchmarkConstructImageManyComponents(b *testing.B) {
	b.ReportAllocs()
	image := getTestImage(200)
	imageFullName := "stackrox.io/srox/test:latest"

	for b.Loop() {
		_, _ = ConstructImage(image, imageFullName)
	}
}

func BenchmarkConstructDeployment(b *testing.B) {
	b.ReportAllocs()
	deployment := &storage.Deployment{
		Name:      "test-deploy",
		Namespace: "default",
		Containers: []*storage.Container{
			{
				Name: "main",
				Config: &storage.ContainerConfig{
					Env: []*storage.ContainerConfig_EnvironmentConfig{
						{EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW, Key: "DATABASE_URL", Value: "postgres://localhost"},
						{EnvVarSource: storage.ContainerConfig_EnvironmentConfig_RAW, Key: "API_KEY", Value: "secret"},
						{EnvVarSource: storage.ContainerConfig_EnvironmentConfig_FIELD, Key: "NODE_NAME", Value: "spec.nodeName"},
					},
				},
			},
		},
	}
	images := []*storage.Image{getTestImage(50)}
	applied := &NetworkPoliciesApplied{}

	for b.Loop() {
		_, _, _ = constructDeployment(deployment, images, applied)
	}
}
