package nodeinventorizer

import (
	"testing"

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
		inComponents []*database.RHELv2Package
		expectedLen  int
	}{
		"nil-inComponents": {
			inComponents: nil,
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
			} else {
				s.Nil(convertedComponents)
			}
		})
	}
}

func (s *NodeInventorizerTestSuite) TestMakeComponentKey() {
	testcases := map[string]struct {
		component *scannerV1.RHELComponent
		expected  string
	}{
		"Full component": {
			component: &scannerV1.RHELComponent{
				Id:      0,
				Name:    "Name",
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: "Name1.2.3x42Mod",
		},
		"Missing part": {
			component: &scannerV1.RHELComponent{
				Id:      0,
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: "1.2.3x42Mod",
		},
		"Internationalized": {
			component: &scannerV1.RHELComponent{
				Id:      0,
				Name:    "日本語",
				Version: "1.2.3",
				Arch:    "x42",
				Module:  "Mod",
			},
			expected: "日本語1.2.3x42Mod",
		},
		"Nil component": {
			component: nil,
			expected:  "",
		},
	}

	for testName, testCase := range testcases {
		s.Run(testName, func() {
			s.Equal(testCase.expected, makeComponentKey(testCase.component))
		})
	}
}
