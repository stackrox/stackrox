package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FieldMetadataValidationSuite struct {
	suite.Suite
}

func TestAllFieldsMetadata(t *testing.T) {
	suite.Run(t, new(FieldMetadataValidationSuite))
}

func (s *FieldMetadataValidationSuite) ValidateAllFieldMetadata() {
	assert.Equal(s.T(), fieldnames.Count(), len(FieldMetadataSingleton().fieldsToQB))
}
