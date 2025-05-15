package s3common

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
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

func TestNewS3Client(t *testing.T) {
	// smoke tests for the constructor logic
	for name, cfg := range map[string]*fakeConfigWrapper{
		"s3 with IAM and empty endpoint": {
			endpoint:   "",
			useIam:     true,
			pluginType: "s3",
			valid:      true,
		},
		"s3 compatible with filled endpoint": {
			endpoint:    "s3compatible.example.com",
			useIam:      false,
			accessKeyID: "access-key-id",
			accessKey:   "access-key",
			pluginType:  "s3compatible",
			valid:       true,
		},
		"broken config": {
			valid: false,
		},
	} {
		t.Run(name, func(it *testing.T) {
			client, err := NewS3Client(cfg)
			if cfg.valid {
				assert.NoError(it, err)
				assert.NotNil(it, client)
			} else {
				assert.ErrorIs(it, err, errox.InvalidArgs)
				assert.Nil(it, client)
			}
		})
	}
}

type fakeConfigWrapper struct {
	endpoint    string
	useIam      bool
	accessKeyID string
	accessKey   string
	pluginType  string
	valid       bool
}

var _ ConfigWrapper = (*fakeConfigWrapper)(nil)

func (w *fakeConfigWrapper) GetUrlStyle() storage.S3URLStyle {
	return storage.S3URLStyle_S3_URL_STYLE_UNSPECIFIED
}

func (w *fakeConfigWrapper) GetEndpoint() string {
	return w.endpoint
}

func (w *fakeConfigWrapper) GetRegion() string {
	return "us-east-1"
}

func (w *fakeConfigWrapper) GetBucket() string {
	return "test-bucket"
}

func (w *fakeConfigWrapper) GetObjectPrefix() string {
	return ""
}

func (w *fakeConfigWrapper) GetUseIam() bool {
	return w.useIam
}

func (w *fakeConfigWrapper) GetAccessKeyId() string {
	return w.accessKeyID
}

func (w *fakeConfigWrapper) GetSecretAccessKey() string {
	return w.accessKey
}

func (w *fakeConfigWrapper) GetName() string {
	return "Test Backup Plugin"
}

func (w *fakeConfigWrapper) GetPluginType() string {
	return w.pluginType
}

func (w *fakeConfigWrapper) GetBackupsToKeep() int32 {
	return 1
}

func (w *fakeConfigWrapper) Validate() error {
	if w.valid {
		return nil
	}
	return errox.InvalidArgs
}
