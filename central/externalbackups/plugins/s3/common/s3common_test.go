package s3common

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointValidation(t *testing.T) {
	testCases := map[string]struct {
		endpoint          string
		sanitizedEndpoint string
		shouldError       bool
	}{
		"with https prefix": {
			endpoint:          "https://play.min.io",
			sanitizedEndpoint: "https://play.min.io",
		},
		"with http prefix": {
			endpoint:          "http://play.min.io",
			sanitizedEndpoint: "http://play.min.io",
		},
		"without prefix": {
			endpoint:          "play.min.io",
			sanitizedEndpoint: "https://play.min.io",
		},
		"invalid URL": {
			endpoint:    "play%min.io",
			shouldError: true,
		},
		"with trailing slash": {
			endpoint:          "https://play.min.io/",
			sanitizedEndpoint: "https://play.min.io",
		},
	}

	for caseName, testCase := range testCases {
		t.Run(caseName, func(t *testing.T) {
			result, err := validateEndpoint(testCase.endpoint)
			if testCase.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.sanitizedEndpoint, result)
		})
	}
}

func TestSortS3Objects(t *testing.T) {
	ts1 := time.Date(2022, 11, 1, 13, 30, 25, 123456789, time.UTC)
	obj1 := types.Object{
		Key:          aws.String("Object 1"),
		LastModified: &ts1,
	}
	ts2 := time.Date(2022, 11, 2, 12, 25, 15, 234567891, time.UTC)
	obj2 := types.Object{
		Key:          aws.String("Object 2"),
		LastModified: &ts2,
	}
	obj3 := types.Object{
		Key: aws.String("Object 3"),
	}

	for name, tc := range map[string]struct {
		input          []types.Object
		expectedOutput []types.Object
	}{
		"Empty input": {
			input:          []types.Object{},
			expectedOutput: []types.Object{},
		},
		"Nil input": {
			input:          nil,
			expectedOutput: nil,
		},
		"Single object": {
			input:          []types.Object{obj1},
			expectedOutput: []types.Object{obj1},
		},
		"Ordered with valid timestamps": {
			input:          []types.Object{obj2, obj1},
			expectedOutput: []types.Object{obj2, obj1},
		},
		"Shuffled with valid timestamps": {
			input:          []types.Object{obj1, obj2},
			expectedOutput: []types.Object{obj2, obj1},
		},
		"Shuffled with nil timestamps": {
			input:          []types.Object{obj1, obj3, obj2},
			expectedOutput: []types.Object{obj2, obj1, obj3},
		},
	} {
		t.Run(name, func(it *testing.T) {
			objects := tc.input
			sortS3Objects(objects)
			assert.Equal(it, tc.expectedOutput, objects)
		})
	}
}
