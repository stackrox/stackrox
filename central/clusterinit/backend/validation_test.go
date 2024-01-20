package backend

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestClusterInitValidation(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(clusterInitValidationTestSuite))
}

type clusterInitValidationTestSuite struct{ suite.Suite }

func (s *clusterInitValidationTestSuite) TestInitBundleNameValidation() {
	validNames := []string{
		"a",
		"A",
		"_",
		"-",
		"0",
		"a.b",
		"a-b-c_0123456789",
		"o_o",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-",
	}
	invalidNames := []string{
		"",
		"a b",
		"x ",
		" 1",
		"a(",
		"/abc",
		"|abc|",
		"*1234567890",
		"foo.bar+",
		"@name",
		"42?",
		"[0-9]",
		"{}",
		"comma,",
	}

	for _, name := range validNames {
		s.Require().NoErrorf(validateName(name), "The name %q failed validation but it is assumed to be valid.", name)
	}

	for _, name := range invalidNames {
		s.Equalf(ErrInvalidInitBundleName, validateName(name), "The name %q validated successfully but it is assumed to be invalid.", name)
	}
}
