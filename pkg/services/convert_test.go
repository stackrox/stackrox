package services

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestServiceTypeToSlugName(t *testing.T) {
	cases := map[storage.ServiceType]string{
		storage.ServiceType_ADMISSION_CONTROL_SERVICE: "admission-control",
		storage.ServiceType_COLLECTOR_SERVICE:         "collector",
		storage.ServiceType_SENSOR_SERVICE:            "sensor",
		storage.ServiceType(99):                       "",
	}

	for svcTy, expectedStr := range cases {
		assert.Equalf(t, expectedStr, ServiceTypeToSlugName(svcTy), "unexpected slug name for service type %v", svcTy)
	}
}
