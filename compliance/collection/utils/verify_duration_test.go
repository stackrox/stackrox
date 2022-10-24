package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func TestVerifyDuration(t *testing.T) {
	suite.Run(t, &verifyDurationTestSuite{})
}

type verifyDurationTestSuite struct {
	suite.Suite
}

func (v *verifyDurationTestSuite) TestVerifyAndUpdateDuration() {
	testCases := map[string]struct {
		setDuration      time.Duration
		expectedDuration time.Duration
	}{
		"valid_duration": {
			setDuration:      time.Hour * 2,
			expectedDuration: time.Hour * 2,
		},
		"zero_duration": {
			setDuration:      time.Duration(0),
			expectedDuration: time.Hour * 4,
		},
		"negative_duration": {
			setDuration:      time.Duration(-5),
			expectedDuration: time.Hour * 4,
		},
	}

	for caseName, testCase := range testCases {
		v.Run(caseName, func() {
			result := VerifyAndUpdateDuration(testCase.setDuration)
			v.Equal(testCase.expectedDuration, result)
		})
	}
}
