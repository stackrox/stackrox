package s3common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stackrox/rox/central/externalbackups/plugins/s3/common/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	sampleDate1 = time.Date(2022, 11, 1, 13, 30, 25, 123456789, time.UTC)

	sampleDate2 = time.Date(2022, 11, 2, 12, 25, 15, 234567891, time.UTC)
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
	obj1 := types.Object{
		Key:          aws.String("Object 1"),
		LastModified: &sampleDate1,
	}
	obj2 := types.Object{
		Key:          aws.String("Object 2"),
		LastModified: &sampleDate2,
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
			endpoint:    "",
			useIam:      true,
			pluginType:  "s3",
			backupCount: 1,
			valid:       true,
		},
		"s3 compatible with filled endpoint": {
			endpoint:    "s3compatible.example.com",
			useIam:      false,
			accessKeyID: "access-key-id",
			accessKey:   "access-key",
			pluginType:  "s3compatible",
			backupCount: 1,
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

func TestPruneBackupsIfNecessary(t *testing.T) {
	var keys []string
	var objects []types.Object
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("Backup %d", i+1)
		var lastModified *time.Time
		if i == 1 {
			lastModified = nil
		} else {
			ts := time.Date(2022, 11, i, 13, 30, 25, 123456789, time.UTC)
			lastModified = &ts
		}
		keys = append(keys, key)
		objects = append(objects, types.Object{
			Key:          aws.String(key),
			LastModified: lastModified,
		})
	}

	for name, tc := range map[string]struct {
		backupsToKeep int32
		savedBackups  []types.Object
		prunedIDs     []string
	}{
		"no backup, nothing to prune": {
			backupsToKeep: 1,
			savedBackups:  make([]types.Object, 0),
			prunedIDs:     []string{},
		},
		"less backups than should be kept, nothing to prune": {
			backupsToKeep: 1,
			savedBackups:  []types.Object{objects[0]},
			prunedIDs:     []string{},
		},
		"some backups to keep, one to prune": {
			backupsToKeep: 2,
			savedBackups:  []types.Object{objects[2], objects[3], objects[1]},
			prunedIDs:     []string{keys[1]},
		},
		"some backups to keep, some to prune": {
			backupsToKeep: 3,
			savedBackups:  objects,
			prunedIDs:     []string{keys[1], keys[0]},
		},
	} {
		t.Run(name, func(it *testing.T) {
			cfg := &fakeConfigWrapper{backupCount: tc.backupsToKeep}
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
			s3Client := s3Common{
				config:        cfg,
				bucket:        "test-bucket",
				now:           time.Now,
				clientWrapper: mockS3,
			}

			listRsp := &s3.ListObjectsV2Output{
				Contents: tc.savedBackups,
			}
			mockS3.EXPECT().
				ListObjects(gomock.Any(), s3Client.prefixKey("backup")).
				Times(1).
				Return(listRsp, nil)

			for _, prunedID := range tc.prunedIDs {
				mockS3.EXPECT().
					Delete(gomock.Any(), prunedID).
					Times(1).
					Return(nil)
			}

			err := s3Client.pruneBackupsIfNecessary(context.Background())
			assert.NoError(it, err)
		})
	}
}

func TestPluginTest(t *testing.T) {
	t.Run("Standard test, no key prefix, normal flow", func(it *testing.T) {
		cfg := &fakeConfigWrapper{}
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		key := "test"
		s3Client := getMockedS3Client(cfg, time.Now, mockS3)
		mockS3.EXPECT().
			Upload(gomock.Any(), key, strings.NewReader(testPayload)).
			Times(1).
			Return(nil)
		mockS3.EXPECT().
			Delete(gomock.Any(), key).
			Times(1).
			Return(nil)
		err := s3Client.Test()
		assert.NoError(it, err)
	})
	t.Run("Standard test, key with prefix, normal flow", func(it *testing.T) {
		cfg := &fakeConfigWrapper{prefix: "backups"}
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		key := "backups/test"
		s3Client := getMockedS3Client(cfg, time.Now, mockS3)
		mockS3.EXPECT().
			Upload(gomock.Any(), key, strings.NewReader(testPayload)).
			Times(1).
			Return(nil)
		mockS3.EXPECT().
			Delete(gomock.Any(), key).
			Times(1).
			Return(nil)
		err := s3Client.Test()
		assert.NoError(it, err)
	})
	t.Run("Test upload failure", func(it *testing.T) {
		cfg := &fakeConfigWrapper{pluginType: "s3"}
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		key := "test"
		s3Client := getMockedS3Client(cfg, time.Now, mockS3)
		uploadErr := "no upload possible"
		mockS3.EXPECT().
			Upload(gomock.Any(), key, strings.NewReader(testPayload)).
			Times(1).
			Return(errors.New(uploadErr))
		err := s3Client.Test()
		expectedError := "creating test object \"test\" in bucket \"test-bucket\""
		assert.ErrorContains(it, err, expectedError)
	})
	t.Run("Test cleanup failure", func(it *testing.T) {
		cfg := &fakeConfigWrapper{
			pluginType: "s3compatible",
			prefix:     "backups",
		}
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		key := "backups/test"
		s3Client := getMockedS3Client(cfg, time.Now, mockS3)
		mockS3.EXPECT().
			Upload(gomock.Any(), key, strings.NewReader(testPayload)).
			Times(1).
			Return(nil)
		cleanupError := &fakeError{}
		mockS3.EXPECT().
			Delete(gomock.Any(), key).
			Times(1).
			Return(cleanupError)
		err := s3Client.Test()
		expectedError := "S3 compatible backup: deleting test object \"backups/test\" from bucket \"test-bucket\" " +
			"(code: fake error code; message: fake error message)"
		assert.ErrorContains(it, err, expectedError)

	})
}

func TestBackup(t *testing.T) {
	t.Run("Test backup success, no key prefix", func(it *testing.T) {
		backupPayload := "test backup success for key without prefix"
		backupBody := io.NopCloser(strings.NewReader(backupPayload))
		key := "backup_2022-11-01T13:30:25.zip"

		cfg := &fakeConfigWrapper{pluginType: "s3"}
		now := func() time.Time { return sampleDate1 }
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		s3Client := getMockedS3Client(cfg, now, mockS3)
		mockS3.EXPECT().Upload(gomock.Any(), key, backupBody).Times(1).Return(nil)
		mockS3.EXPECT().ListObjects(gomock.Any(), "backup").Times(1).Return(&s3.ListObjectsV2Output{}, nil)

		err := s3Client.Backup(backupBody)
		assert.NoError(it, err)
	})
	t.Run("Test backup success, key with prefix", func(it *testing.T) {
		backupPayload := "test backup success for key with prefix"
		backupBody := io.NopCloser(strings.NewReader(backupPayload))
		key := "backups/backup_2022-11-02T12:25:15.zip"

		cfg := &fakeConfigWrapper{prefix: "backups"}
		now := func() time.Time { return sampleDate2 }
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		s3Client := getMockedS3Client(cfg, now, mockS3)
		mockS3.EXPECT().Upload(gomock.Any(), key, backupBody).Times(1).Return(nil)
		mockS3.EXPECT().ListObjects(gomock.Any(), "backups/backup").Times(1).Return(&s3.ListObjectsV2Output{}, nil)

		err := s3Client.Backup(backupBody)
		assert.NoError(it, err)
	})
	t.Run("Test backup upload failure", func(it *testing.T) {
		backupPayload := "test backup upload failure payload"
		backupBody := io.NopCloser(strings.NewReader(backupPayload))
		key := "backup_2022-11-01T13:30:25.zip"

		cfg := &fakeConfigWrapper{pluginType: "s3"}
		now := func() time.Time { return sampleDate1 }
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		mockS3 := mocks.NewMocks3Wrapper(mockCtrl)
		s3Client := getMockedS3Client(cfg, now, mockS3)
		mockS3.EXPECT().Upload(gomock.Any(), key, backupBody).Times(1).Return(&fakeError{})

		err := s3Client.Backup(backupBody)
		expectedErrorMessage := "S3 backup: creating backup in bucket \"test-bucket\" with key \"backup_2022-11-01T13:30:25.zip\" " +
			"(code: fake error code; message: fake error message)"
		assert.ErrorContains(it, err, expectedErrorMessage)
	})
}

// region helpers

func getMockedS3Client(cfg ConfigWrapper, now func() time.Time, awsClient s3Wrapper) *s3Common {
	return &s3Common{
		config:        cfg,
		bucket:        "test-bucket",
		now:           now,
		clientWrapper: awsClient,
	}
}

type fakeConfigWrapper struct {
	endpoint    string
	useIam      bool
	accessKeyID string
	accessKey   string
	pluginType  string
	prefix      string
	backupCount int32
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
	return w.prefix
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
	return w.backupCount
}

func (w *fakeConfigWrapper) Validate() error {
	if w.valid {
		return nil
	}
	return errox.InvalidArgs
}

type fakeError struct{}

var _ error = (*fakeError)(nil)
var _ smithy.APIError = (*fakeError)(nil)

func (e *fakeError) Error() string {
	return "error"
}

func (e *fakeError) ErrorCode() string {
	return "fake error code"
}

func (e *fakeError) ErrorMessage() string {
	return "fake error message"
}

func (e *fakeError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultUnknown
}

// endregion helpers
