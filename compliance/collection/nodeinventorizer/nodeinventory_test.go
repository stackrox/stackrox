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
		inComponents  []*database.RHELv2Package
		outComponents []*scannerV1.RHELComponent
		expectedLen   int
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
			convertedComponents := convertRHELComponents(mockComponents)
			if testCase.inComponents != nil {
				convertedIDs := make([]int64, 0, len(convertedComponents))
				for _, entry := range convertedComponents {
					convertedIDs = append(convertedIDs, entry.Id)
				}
				s.Equal(testCase.expectedLen, len(convertedComponents))
			} else {
				s.Nil(convertedComponents)
			}
		})
	}
}

func (s *NodeInventorizerTestSuite) TestEqualRHELv2Packages() {
	testcases := map[string]struct {
		a        *database.RHELv2Package
		b        *database.RHELv2Package
		expected bool
	}{
		"equal components": {
			a: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			b: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			expected: true,
		},
		"empty comparable": {
			a: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			b: &database.RHELv2Package{
				Name:    "",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			expected: false,
		},
		"missing comparable": {
			a: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			b: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
			},
			expected: false,
		},
		"missing comparables": {
			a: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
			},
			b: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
			},
			expected: true,
		},
		"diff comparable": {
			a: &database.RHELv2Package{
				Name:   "tes",
				Arch:   "x86",
				Module: "mod",
			},
			b: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
			},
			expected: false,
		},
		"capitalized components": {
			a: &database.RHELv2Package{
				Name:    "Tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			b: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "moD",
			},
			expected: false,
		},
		"Nil component": {
			a: &database.RHELv2Package{
				Name:    "tes",
				Version: "1.0",
				Arch:    "x86",
				Module:  "mod",
			},
			b:        nil,
			expected: false,
		},
		"Nil components": {
			a:        nil,
			b:        nil,
			expected: false,
		},
	}

	for testName, testCase := range testcases {
		s.Run(testName, func() {
			s.Equal(testCase.expected, equalRHELv2Packages(testCase.a, testCase.b))
		})
	}
}
