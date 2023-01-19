package nodeinventorizer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/scanner/database"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestNodeInventorizer(t *testing.T) {
	suite.Run(t, new(NodeInventorizerTestSuite))
}

type NodeInventorizerTestSuite struct {
	suite.Suite
}

func (s *NodeInventorizerTestSuite) TestConvertRHELComponentIDs() {
	testCases := map[string]struct {
		inComponents  []*database.RHELv2Package
		outComponents []*storage.NodeInventory_Components_RHELComponent
		expectedLen   int
	}{
		"nil-inComponents": {
			inComponents:  nil,
			outComponents: make([]*storage.NodeInventory_Components_RHELComponent, 0),
		},
		"one-component": {
			inComponents: []*database.RHELv2Package{
				{
					Name:    "zlib",
					Version: "1.2.11-16.el8_2",
					Arch:    "x86_64",
					ExecutableToDependencies: database.StringToStringsMap{
						"/usr/lib64/libz.so.1":      {},
						"/usr/lib64/libz.so.1.2.11": {},
					},
				},
			},
			outComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:        0,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
				},
			},
			expectedLen: 1,
		},
		"multi-component": {
			inComponents: []*database.RHELv2Package{
				{
					Name:    "zlib",
					Version: "1.2.11-16.el8_2",
					Arch:    "x86_64",
					ExecutableToDependencies: database.StringToStringsMap{
						"/usr/lib64/libz.so.1":      {},
						"/usr/lib64/libz.so.1.2.11": {},
					},
				},
				{
					Name:    "redhat-release",
					Version: "8.3-1.0.el8",
					Arch:    "x86_64",
				},
			},
			outComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:        0,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
				},
				{
					Id:        1,
					Name:      "redhat-release",
					Namespace: "MockDist",
					Version:   "8.3-1.0.el8",
					Arch:      "x86_64",
				},
			},
			expectedLen: 2,
		},
		"collision-component": {
			inComponents: []*database.RHELv2Package{
				{
					Name:    "redhat-release",
					Version: "8.3-1.0.el8",
					Arch:    "x86_64",
				},
				{
					Name:    "redhat-release",
					Version: "8.3-1.0.el8",
					Arch:    "x86_64",
				},
			},
			outComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:        0,
					Name:      "redhat-release",
					Namespace: "MockDist",
					Version:   "8.3-1.0.el8",
					Arch:      "x86_64",
				},
			},
			expectedLen: 1,
		},
	}
	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			mockComponents := &database.RHELv2Components{
				Dist:     "MockDist",
				CPEs:     nil,
				Packages: testCase.inComponents,
			}
			convertedComponents := convertAndDedupRHELComponents(mockComponents)
			if testCase.inComponents != nil {
				s.Equal(testCase.expectedLen, len(convertedComponents))
				s.ElementsMatch(testCase.outComponents, convertedComponents)
			} else {
				s.Nil(convertedComponents)
			}
		})
	}
}

func (s *NodeInventorizerTestSuite) TestMakeComponentKey() {
	testcases := map[string]struct {
		component *storage.NodeInventory_Components_RHELComponent
		expected  string
	}{
		"Full component": {
			component: &storage.NodeInventory_Components_RHELComponent{
				Id:      0,
				Name:    "Name",
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: "Name:1.2.3:x42:Mod",
		},
		"Missing part": {
			component: &storage.NodeInventory_Components_RHELComponent{
				Id:      0,
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: ":1.2.3:x42:Mod",
		},
		"Internationalized": {
			component: &storage.NodeInventory_Components_RHELComponent{
				Id:      0,
				Name:    "日本語",
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: "日本語:1.2.3:x42:Mod",
		},
	}

	for testName, testCase := range testcases {
		s.Run(testName, func() {
			s.Equal(testCase.expected, makeComponentKey(testCase.component))
		})
	}
}

func (s *NodeInventorizerTestSuite) TestConvertExecutable() {
	testcases := map[string]struct {
		exe      []*scannerV1.Executable
		expected []*storage.NodeInventory_Components_RHELComponent_Executable
	}{
		"RequiredFeatures not empty": {
			exe: []*scannerV1.Executable{
				{
					Path: "/root/1",
					RequiredFeatures: []*scannerV1.FeatureNameVersion{
						{
							Name:    "name1",
							Version: "version1",
						},
					},
				},
			},
			expected: []*storage.NodeInventory_Components_RHELComponent_Executable{
				{
					Path: "/root/1",
					RequiredFeatures: []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
						{
							Name:    "name1",
							Version: "version1",
						},
					},
				},
			},
		},
		"RequiredFeatures empty": {
			exe: []*scannerV1.Executable{
				{
					Path:             "/root/1",
					RequiredFeatures: []*scannerV1.FeatureNameVersion{},
				},
			},
			expected: []*storage.NodeInventory_Components_RHELComponent_Executable{
				{
					Path:             "/root/1",
					RequiredFeatures: []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{},
				},
			},
		},
		"RequiredFeatures nil": {
			exe: []*scannerV1.Executable{
				{
					Path:             "/root/1",
					RequiredFeatures: nil,
				},
			},
			expected: []*storage.NodeInventory_Components_RHELComponent_Executable{
				{
					Path:             "/root/1",
					RequiredFeatures: nil,
				},
			},
		},
	}

	for testName, testCase := range testcases {
		s.Run(testName, func() {
			for i, got := range convertExecutables(testCase.exe) {
				s.Equal(testCase.expected[i].GetPath(), got.GetPath())
				s.Equal(testCase.expected[i].GetRequiredFeatures(), got.GetRequiredFeatures())
			}
		})
	}
}
