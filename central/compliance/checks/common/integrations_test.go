package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"go.uber.org/mock/gomock"
)

type prefixMatcher struct {
	prefix string
}

func (p *prefixMatcher) Match(image *storage.ImageName) bool {
	return strings.HasPrefix(image.GetFullName(), p.prefix)
}

type deployment struct {
	name   string
	images []string
}

func constructDeployments(lightDeps []deployment) map[string]*storage.Deployment {
	outputDeps := make(map[string]*storage.Deployment)

	for _, lightDep := range lightDeps {
		containers := make([]*storage.Container, 0, len(lightDep.images))
		for _, img := range lightDep.images {
			containers = append(containers, &storage.Container{Image: &storage.ContainerImage{Name: &storage.ImageName{FullName: img}}})
		}
		outputDeps[lightDep.name] = &storage.Deployment{
			Name:       lightDep.name,
			Containers: containers,
		}
	}
	return outputDeps
}

func TestCheckAllMatchingIntegrations(t *testing.T) {
	validMatcher := &prefixMatcher{"valid"}
	nothingMatcher := &prefixMatcher{"NOTHINGWILLHAVETHISPREFIX"}

	for _, testCase := range []struct {
		desc        string
		deployments []deployment
		registries  []framework.ImageMatcher
		scanners    []framework.ImageMatcher
		shouldPass  bool
	}{
		{
			desc: "Single deployment, valid image, matching integrations",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
			},
			registries: []framework.ImageMatcher{validMatcher},
			scanners:   []framework.ImageMatcher{validMatcher},
			shouldPass: true,
		},
		{
			desc: "Single deployment, valid image, but no matching scanner integration",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
			},
			registries: []framework.ImageMatcher{validMatcher},
			shouldPass: false,
		},
		{
			desc: "Single deployment, valid image, but no matching registry integration",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
			},
			scanners:   []framework.ImageMatcher{validMatcher},
			shouldPass: false,
		},
		{
			desc: "Single deployment, valid image, but bad matching scanner integration",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
			},
			registries: []framework.ImageMatcher{validMatcher},
			scanners:   []framework.ImageMatcher{nothingMatcher},
			shouldPass: false,
		},
		{
			desc: "Single deployment, valid image, one matching integration at least",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
			},
			registries: []framework.ImageMatcher{validMatcher, nothingMatcher},
			scanners:   []framework.ImageMatcher{nothingMatcher, validMatcher},
			shouldPass: true,
		},
		{
			desc: "Single deployment, valid image and invalid image, one matching integration at least",
			deployments: []deployment{
				{"blah", []string{"validImage", "invalidimage"}},
			},
			registries: []framework.ImageMatcher{validMatcher, nothingMatcher},
			scanners:   []framework.ImageMatcher{nothingMatcher, validMatcher},
			shouldPass: false,
		},
		{
			desc: "Multiple deployments, valid images",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
				{"blah2", []string{"validImage2"}},
			},
			registries: []framework.ImageMatcher{validMatcher, nothingMatcher},
			scanners:   []framework.ImageMatcher{nothingMatcher, validMatcher},
			shouldPass: true,
		},
		{
			desc: "Multiple deployments, only one with valid image images",
			deployments: []deployment{
				{"blah", []string{"validImage"}},
				{"blah2", []string{"superinvalidimage"}},
			},
			registries: []framework.ImageMatcher{validMatcher, nothingMatcher},
			scanners:   []framework.ImageMatcher{nothingMatcher, validMatcher},
			shouldPass: false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			mockData.EXPECT().Deployments().Return(constructDeployments(c.deployments))
			mockData.EXPECT().RegistryIntegrations().Return(c.registries)
			mockData.EXPECT().ScannerIntegrations().Return(c.scanners)
			CheckAllDeployedImagesHaveMatchingIntegrations(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}

}
