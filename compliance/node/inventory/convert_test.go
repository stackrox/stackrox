package inventory

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestInventoryConvert(t *testing.T) {
	suite.Run(t, new(inventoryConvertTestSuite))
}

type inventoryConvertTestSuite struct {
	suite.Suite
}

func (s *inventoryConvertTestSuite) TestToNodeInventory() {
	in := &scannerV1.GetNodeInventoryResponse{
		NodeName: "testme",
		Components: &scannerV1.Components{
			Namespace: "rhcos:testme",
			RhelComponents: []*scannerV1.RHELComponent{
				{
					Id:        int64(42),
					Name:      "libksba",
					Namespace: "rhel:8",
					Version:   "1.3.5-7.el8",
					Arch:      "x86_64",
				},
			},
			RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
		},
		Notes: []scannerV1.Note{scannerV1.Note_OS_CVES_STALE},
	}

	actual := ToNodeInventory(in)

	s.Equal(uuid.Nil.String(), actual.GetNodeId(), "NodeId must be Nil in NodeInventory per convention")
	s.Equal(in.GetNodeName(), actual.GetNodeName())
	s.Equal(len(in.GetComponents().GetRhelComponents()), len(actual.GetComponents().GetRhelComponents()))
	s.Equal(in.GetComponents().GetRhelContentSets(), actual.GetComponents().GetRhelContentSets())
	s.Equal([]storage.NodeInventory_Note{storage.NodeInventory_OS_CVES_STALE}, actual.GetNotes())
}

func (s *inventoryConvertTestSuite) TestToStorageComponents() {
	testCases := map[string]struct {
		inComponent  *scannerV1.Components
		outComponent *storage.NodeInventory_Components
	}{
		"set component": {
			inComponent: &scannerV1.Components{
				Namespace: "rhcos:testme",
				RhelComponents: []*scannerV1.RHELComponent{
					{
						Id:        int64(1),
						Name:      "libksba",
						Namespace: "rhel:8",
						Version:   "1.3.5-7.el8",
						Arch:      "x86_64",
						Module:    "",
						Cpes:      []string{},
						AddedBy:   "",
					},
					{
						Id:        int64(2),
						Name:      "tar",
						Namespace: "rhel:8",
						Version:   "1.27.1.el8",
						Arch:      "x86_64",
						Module:    "",
						Cpes:      []string{},
						AddedBy:   "",
					},
				},
				RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
			},
			outComponent: storage.NodeInventory_Components_builder{
				Namespace: "rhcos:testme",
				RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
					storage.NodeInventory_Components_RHELComponent_builder{
						Id:        int64(1),
						Name:      "libksba",
						Namespace: "rhel:8",
						Version:   "1.3.5-7.el8",
						Arch:      "x86_64",
						Module:    "",
					}.Build(),
					storage.NodeInventory_Components_RHELComponent_builder{
						Id:        int64(2),
						Name:      "tar",
						Namespace: "rhel:8",
						Version:   "1.27.1.el8",
						Arch:      "x86_64",
						Module:    "",
					}.Build(),
				},
				RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
			}.Build(),
		},
		"nil component": {
			inComponent:  nil,
			outComponent: nil,
		},
		"empty component and namespace": {
			inComponent: &scannerV1.Components{
				Namespace:          "",
				OsComponents:       nil,
				RhelComponents:     nil,
				LanguageComponents: nil,
				RhelContentSets:    nil,
			},
			outComponent: storage.NodeInventory_Components_builder{
				Namespace:       "",
				RhelComponents:  nil,
				RhelContentSets: nil,
			}.Build(),
		},
	}
	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			convertedComponent := toStorageComponents(testCase.inComponent)
			if testCase.inComponent != nil {
				protoassert.Equal(s.T(), testCase.outComponent, convertedComponent)
			} else {
				s.Nil(convertedComponent)
			}
		})
	}
}

func (s *inventoryConvertTestSuite) TestConvertRHELComponents() {
	testCases := map[string]struct {
		inComponents  []*scannerV1.RHELComponent
		outComponents []*storage.NodeInventory_Components_RHELComponent
	}{
		"nil-inComponents": {
			inComponents:  nil,
			outComponents: make([]*storage.NodeInventory_Components_RHELComponent, 0),
		},
		"one-component": {
			inComponents: []*scannerV1.RHELComponent{
				{
					Id:        42,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
					Module:    "",
				},
			},
			outComponents: []*storage.NodeInventory_Components_RHELComponent{
				storage.NodeInventory_Components_RHELComponent_builder{
					Id:        42,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
					Module:    "",
				}.Build(),
			},
		},
		"multi-component": {
			inComponents: []*scannerV1.RHELComponent{
				{
					Id:        10,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
				},
				{
					Id:        11,
					Name:      "redhat-release",
					Namespace: "MockDist",
					Version:   "8.3-1.0.el8",
					Arch:      "x86_64",
				},
			},
			outComponents: []*storage.NodeInventory_Components_RHELComponent{
				storage.NodeInventory_Components_RHELComponent_builder{
					Id:        10,
					Name:      "zlib",
					Namespace: "MockDist",
					Version:   "1.2.11-16.el8_2",
					Arch:      "x86_64",
				}.Build(),
				storage.NodeInventory_Components_RHELComponent_builder{
					Id:        11,
					Name:      "redhat-release",
					Namespace: "MockDist",
					Version:   "8.3-1.0.el8",
					Arch:      "x86_64",
				}.Build(),
			},
		},
	}
	for caseName, testCase := range testCases {
		s.Run(caseName, func() {
			convertedComponents := convertRHELComponents(testCase.inComponents)
			if testCase.inComponents != nil {
				s.Equal(len(testCase.inComponents), len(convertedComponents))
				protoassert.ElementsMatch(s.T(), testCase.outComponents, convertedComponents)
			} else {
				s.Nil(convertedComponents)
			}
		})
	}
}

func (s *inventoryConvertTestSuite) TestConvertExecutables() {
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
				storage.NodeInventory_Components_RHELComponent_Executable_builder{
					Path: "/root/1",
					RequiredFeatures: []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{
						storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion_builder{
							Name:    "name1",
							Version: "version1",
						}.Build(),
					},
				}.Build(),
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
				storage.NodeInventory_Components_RHELComponent_Executable_builder{
					Path:             "/root/1",
					RequiredFeatures: []*storage.NodeInventory_Components_RHELComponent_Executable_FeatureNameVersion{},
				}.Build(),
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
				storage.NodeInventory_Components_RHELComponent_Executable_builder{
					Path:             "/root/1",
					RequiredFeatures: nil,
				}.Build(),
			},
		},
	}

	for testName, testCase := range testcases {
		s.Run(testName, func() {
			for i, got := range convertExecutables(testCase.exe) {
				s.Equal(testCase.expected[i].GetPath(), got.GetPath())
				protoassert.SlicesEqual(s.T(), testCase.expected[i].GetRequiredFeatures(), got.GetRequiredFeatures())
			}
		})
	}
}

func (s *inventoryConvertTestSuite) TestConvertNotes() {
	in := []scannerV1.Note{
		scannerV1.Note_OS_CVES_UNAVAILABLE,
		scannerV1.Note_OS_CVES_STALE,
		scannerV1.Note_LANGUAGE_CVES_UNAVAILABLE,
		scannerV1.Note_CERTIFIED_RHEL_SCAN_UNAVAILABLE,
	}

	actual := convertNotes(in)

	s.Equal(len(in), len(actual))
	s.Contains(actual, storage.NodeInventory_OS_CVES_UNAVAILABLE)
	s.Contains(actual, storage.NodeInventory_OS_CVES_STALE)
	s.Contains(actual, storage.NodeInventory_LANGUAGE_CVES_UNAVAILABLE)
	s.Contains(actual, storage.NodeInventory_CERTIFIED_RHEL_SCAN_UNAVAILABLE)
}

func (s *inventoryConvertTestSuite) TestConvertNotesOnNil() {
	actual := convertNotes(nil)

	s.Nil(actual)
}
