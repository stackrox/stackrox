package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FieldMetadataValidationSuite struct {
	suite.Suite
}

func TestAllFieldsMetadata(t *testing.T) {
	t.Setenv(features.CVEFixTimestampCriteria.EnvVar(), "true")
	t.Setenv(features.SensitiveFileActivity.EnvVar(), "true")
	ResetFieldMetadataSingleton(t)
	suite.Run(t, new(FieldMetadataValidationSuite))
}

func (s *FieldMetadataValidationSuite) TestValidateAllFieldMetadata() {
	assert.Equal(s.T(), fieldnames.Count(), len(FieldMetadataSingleton().fieldsToQB))
}
