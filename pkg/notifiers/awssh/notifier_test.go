package awssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mocks/github.com/aws/aws-sdk-go/service/securityhub/securityhubiface/mocks"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// A LightAlert is a lightweight alert struct that is very convenient to define in tests.
type LightAlert struct {
	id    string
	state storage.ViolationState
}

func (l *LightAlert) convert() *storage.Alert {
	return &storage.Alert{
		Id: stringutils.OrDefault(l.id, uuid.NewV4().String()),
		Policy: &storage.Policy{
			Id:          uuid.NewV4().String(),
			Name:        "example policy",
			Description: "Some random description",
			Severity:    storage.Severity_HIGH_SEVERITY,
		},
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          uuid.NewV4().String(),
				Name:        "example deployment",
				Namespace:   "example namespace",
				ClusterId:   uuid.NewV4().String(),
				ClusterName: "example cluster",
				Containers: []*storage.Alert_Deployment_Container{
					{
						Name: "example container",
						Image: &storage.ContainerImage{
							Id: uuid.NewV4().String(),
							Name: &storage.ImageName{
								FullName: "registry/path/to/image:tag",
							},
						},
					},
				},
			},
		},
		Violations: []*storage.Alert_Violation{
			{Message: "one"},
			{Message: "two"},
			{Message: "three"},
			{Message: "https://www.stackrox.com"},
		},
		FirstOccurred: types.TimestampNow(),
		Time:          types.TimestampNow(),
		State:         l.state,
	}
}

type BatchSizeMatcher struct {
	size int
}

func (b *BatchSizeMatcher) Matches(in interface{}) bool {
	inputBatch := in.(*securityhub.BatchImportFindingsInput)
	return len(inputBatch.Findings) == b.size
}

func (b *BatchSizeMatcher) String() string {
	return fmt.Sprintf("Matches BatchImportFindingsInput with len(Findings) == %d", b.size)
}

func batchOfSize(n int) *BatchSizeMatcher {
	return &BatchSizeMatcher{size: n}
}

// This function converts a BatchImportFindingsInput to a BatchImportFindingsOutput,
// with all notifications being successful.
func mockBatchImportFindingsWithContext() func(_ aws.Context, input *securityhub.BatchImportFindingsInput, _ ...request.Option) (*securityhub.BatchImportFindingsOutput, error) {
	return mockBatchImportFindingsWithContextWithFailures(0)
}

// This function converts a BatchImportFindingsInput to a BatchImportFindingsOutput,
// setting the appropriate number of successes and (if non-zero) failures.
func mockBatchImportFindingsWithContextWithFailures(failures int) func(_ aws.Context, input *securityhub.BatchImportFindingsInput, _ ...request.Option) (*securityhub.BatchImportFindingsOutput, error) {
	return func(_ aws.Context, input *securityhub.BatchImportFindingsInput, _ ...request.Option) (*securityhub.BatchImportFindingsOutput, error) {
		failedFindings := make([]*securityhub.ImportFindingsError, 0, failures)
		if failures > len(input.Findings) {
			failures = len(input.Findings)
		}
		for _, finding := range input.Findings[:failures] {
			errorCode := "Mocked BatchImportFindings error code"
			errorMessage := "Mocked BatchImportFindings error message"
			failedFindings = append(failedFindings, &securityhub.ImportFindingsError{
				ErrorCode:    &errorCode,
				ErrorMessage: &errorMessage,
				Id:           finding.Id,
			})
		}
		failedCount := int64(failures)
		successCount := int64(len(input.Findings) - failures)

		return &securityhub.BatchImportFindingsOutput{
			FailedCount:    &failedCount,
			FailedFindings: failedFindings,
			SuccessCount:   &successCount,
		}, nil
	}
}

func TestNotifier(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(notifierTestSuite))
}

type notifierTestSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	mockSecurityHub *mocks.MockSecurityHubAPI

	n *notifier
}

func (s *notifierTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSecurityHub = mocks.NewMockSecurityHubAPI(s.mockCtrl)
}

func (s *notifierTestSuite) runNotifier(ctx context.Context, configuration configuration) {
	s.n = &notifier{
		configuration: configuration,
		securityHub:   s.mockSecurityHub,
		account:       "",
		arn:           "",
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
		// stoppedSig intentionally omitted - zero value is "already triggered"
	}

	go func() {
		switch err := s.n.run(ctx); err {
		case nil, context.Canceled, context.DeadlineExceeded:
			log.Debug("ceasing notifier operation", logging.Err(err))
		default:
			require.NoError(s.T(), err)
		}
	}()

	s.n.waitForInitDone()
}

func (s *notifierTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *notifierTestSuite) TestNotifierSendsBatchWhenFull() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.maxBatchSize = 2
	s.runNotifier(ctx, configuration)

	// TODO(evan): Refine this matcher to not catch notifier.Test() messages
	// (and, though it uses a different API method, notifier.sendHeartbeat())
	s.mockSecurityHub.EXPECT().
		BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
		DoAndReturn(mockBatchImportFindingsWithContext()).Times(1)

	for _, id := range []string{"1", "2"} {
		s.n.alertCh <- (&LightAlert{id: id}).convert()
	}

	s.True(concurrency.WaitWithTimeout(&s.n.stoppedSig, 1500*time.Millisecond), "notifier did not shut down in time")
}

func (s *notifierTestSuite) TestNotifierSendsBatchWhenFreshnessTimerTicks() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.maxBatchSize = 2
	configuration.minUploadDelay = 50 * time.Millisecond  // Don't throttle
	configuration.maxUploadDelay = 300 * time.Millisecond // Send "freshness" batch before context is done
	s.runNotifier(ctx, configuration)

	// TODO(evan): Refine this matcher to not catch notifier.Test() messages
	// (and, though it uses a different API method, notifier.sendHeartbeat())
	s.mockSecurityHub.EXPECT().
		BatchImportFindingsWithContext(gomock.Any(), batchOfSize(1)).
		DoAndReturn(mockBatchImportFindingsWithContext()).Times(1)

	for _, id := range []string{"1"} {
		s.n.alertCh <- (&LightAlert{id: id}).convert()
	}

	s.True(concurrency.WaitWithTimeout(&s.n.stoppedSig, 1500*time.Millisecond), "notifier did not shut down in time")
}

func (s *notifierTestSuite) TestFailedNotificationsAreRetried() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.maxBatchSize = 2
	configuration.minUploadDelay = 50 * time.Millisecond  // Don't throttle
	configuration.maxUploadDelay = 100 * time.Millisecond // Retry before context is done
	s.runNotifier(ctx, configuration)

	// TODO(evan): Refine this matcher to not catch notifier.Test() messages
	// (and, though it uses a different API method, notifier.sendHeartbeat())
	s.mockSecurityHub.EXPECT().
		BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
		DoAndReturn(mockBatchImportFindingsWithContextWithFailures(2)).Times(1).
		DoAndReturn(mockBatchImportFindingsWithContext()).Times(1)

	for _, id := range []string{"1", "2"} {
		s.n.alertCh <- (&LightAlert{id: id}).convert()
	}

	s.True(concurrency.WaitWithTimeout(&s.n.stoppedSig, 1500*time.Millisecond), "notifier did not shut down in time")
}

func (s *notifierTestSuite) TestFailedNotificationsAreRetriedPartialBatchFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.maxBatchSize = 2
	configuration.minUploadDelay = 25 * time.Millisecond  // Don't throttle
	configuration.maxUploadDelay = 100 * time.Millisecond // Retry before context is done
	s.runNotifier(ctx, configuration)

	// TODO(evan): Refine this matcher to not catch notifier.Test() messages
	// (and, though it uses a different API method, notifier.sendHeartbeat())
	gomock.InOrder(
		s.mockSecurityHub.EXPECT().
			BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
			// Fail one notification from the first batch (ID 1 or 2), the other succeeds.
			DoAndReturn(mockBatchImportFindingsWithContextWithFailures(1)).Times(1),
		s.mockSecurityHub.EXPECT().
			BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
			// Fail one notification from the second batch (retried 1 or 2 and new alert 3)
			DoAndReturn(mockBatchImportFindingsWithContextWithFailures(1)).Times(1),
		s.mockSecurityHub.EXPECT().
			BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
			// Succeed both remaining alerts (retried 1, 2, or 3 and new alert 4).
			DoAndReturn(mockBatchImportFindingsWithContext()).Times(1))

	for _, id := range []string{"1", "2", "3", "4"} {
		s.n.alertCh <- (&LightAlert{id: id}).convert()
	}

	s.True(concurrency.WaitWithTimeout(&s.n.stoppedSig, 1500*time.Millisecond), "notifier did not shut down in time")
}

func (s *notifierTestSuite) TestThrottling() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.maxBatchSize = 2
	// Allow at most 2 batches to be sent. The token bucket starts full, then will require 600ms
	// before sending again. No matter how many alerts we queue we should only upload twice in 1s.
	configuration.minUploadDelay = 600 * time.Millisecond // Allow at most 2 batches to be sent
	configuration.maxUploadDelay = 50 * time.Millisecond  // Try to send a batch often, get rate limited
	s.runNotifier(ctx, configuration)

	// TODO(evan): Refine this matcher to not catch notifier.Test() messages
	// (and, though it uses a different API method, notifier.sendHeartbeat())
	s.mockSecurityHub.EXPECT().
		BatchImportFindingsWithContext(gomock.Any(), batchOfSize(2)).
		DoAndReturn(mockBatchImportFindingsWithContext()).Times(2)

	for _, id := range []string{"1", "2", "3", "4", "5", "6"} {
		s.n.alertCh <- (&LightAlert{id: id}).convert()
	}

	s.True(concurrency.WaitWithTimeout(&s.n.stoppedSig, 1500*time.Millisecond), "notifier did not shut down in time")
}

func configFromEnv() (*storage.AWSSecurityHub, error) {
	result := storage.AWSSecurityHub{}

	if v, ok := os.LookupEnv("NOTIFIER_AWS_ACCOUNT_ID"); ok {
		result.AccountId = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_ACCOUNT_ID in env")
	}

	if v, ok := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_REGION"); ok {
		result.Region = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_REGION in env")
	}

	if v, ok := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_ACCESS_KEY_ID"); ok {
		result.Credentials = &storage.AWSSecurityHub_Credentials{
			AccessKeyId: v,
		}
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_ACCESS_KEY_ID in env")
	}

	if v, ok := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_SECRET_ACCESS_KEY"); ok {
		result.Credentials.SecretAccessKey = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_SECRET_ACCESS_KEY in env")
	}

	return &result, nil
}

// TestNotifierCreationFromEnvAndTest exercises very basic functionality. In fact,
// it does not even mutate a security hub instance but rather checks that
// low-level configuration and permissions are correct.
func TestNotifierCreationFromEnvAndTest(t *testing.T) {
	config, err := configFromEnv()
	if err != nil {
		if testutils.IsRunningInCI() {
			// TODO(tvoss): Fail with an error once we have credentials for CI runs.
			t.Skip("failed to load config from env", err)
		} else {
			t.Skip("failed to load config from env", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configuration := defaultConfiguration
	configuration.descriptor = &storage.Notifier{
		Config: &storage.Notifier_AwsSecurityHub{
			AwsSecurityHub: config,
		},
	}
	configuration.maxBatchSize = 3
	configuration.minUploadDelay = 1 * time.Second
	configuration.maxUploadDelay = 15 * time.Second
	configuration.canceler = cancel

	errCh := make(chan error)

	notifier, err := newNotifier(configuration)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, notifier.Close(ctx))
	}()

	go func() {
		errCh <- notifier.run(ctx)
	}()

	notifier.waitForInitDone()

	require.NoError(t, notifier.Test(ctx))
	require.NoError(t, notifier.AlertNotify(ctx, (&LightAlert{state: storage.ViolationState_ACTIVE}).convert()))
	require.NoError(t, notifier.AlertNotify(ctx, (&LightAlert{state: storage.ViolationState_SNOOZED}).convert()))
	require.NoError(t, notifier.AlertNotify(ctx, (&LightAlert{state: storage.ViolationState_RESOLVED}).convert()))

	require.Equal(t, context.DeadlineExceeded, <-errCh)
}
