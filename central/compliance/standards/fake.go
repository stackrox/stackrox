package standards

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

var (
	errNotFound = fmt.Errorf("not found")

	standards = []*v1.ComplianceStandardMetadata{
		{
			Id:          "fake",
			Name:        "Fake",
			Description: "Fake compliance standard",
		},
	}

	groups = []*v1.ComplianceControlGroup{
		{
			Id:          "fake-1",
			StandardId:  "fake",
			Name:        "Frobotzing",
			Description: "Everything must be frobotzed. This standard requires that all items must be frobotzed.",
		},
	}

	controls = []*v1.ComplianceControl{
		{
			Id:          "fake-1.1.1",
			StandardId:  "fake",
			GroupId:     "fake-1",
			Name:        "Fake 1.1.1: Deployments",
			Description: "Your deployments must be frobotzed.",
		},
		{
			Id:          "fake-1.1.2",
			StandardId:  "fake",
			GroupId:     "fake-1",
			Name:        "Fake 1.1.2: Nodes",
			Description: "Your nodes must be frobotzed.",
		},
	}
)

type fake struct{}

func (f *fake) Standards() ([]*v1.ComplianceStandardMetadata, error) {
	return standards, nil
}

func (f *fake) Standard(id string) (*v1.ComplianceStandardMetadata, bool, error) {
	for _, v := range standards {
		if v.GetId() == id {
			return v, true, nil
		}
	}
	return nil, false, nil
}

func (f *fake) Controls(standardID string) ([]*v1.ComplianceControl, error) {
	var out []*v1.ComplianceControl
	for _, v := range controls {
		if v.GetStandardId() == standardID {
			out = append(out, v)
		}
	}
	if out == nil {
		return nil, errNotFound
	}
	return out, nil
}

// Fake returns a standards object that contains one fake standard
func Fake() Standards {
	return &fake{}
}
