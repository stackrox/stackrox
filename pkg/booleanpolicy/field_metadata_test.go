package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FieldMetadataValidationSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
}

func (s *FieldMetadataValidationSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *FieldMetadataValidationSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()
}

func TestAllFieldsMetadata(t *testing.T) {
	suite.Run(t, new(FieldMetadataValidationSuite))
}

func (s *FieldMetadataValidationSuite) ValidateAllFieldMetadata() {
	assert.Equal(s.T(), fieldnames.Count(), len(FieldMetadataSingleton().fieldsToQB))
}
