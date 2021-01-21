package awssh

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func configFromEnv() (*storage.AWSSecurityHub, error) {
	result := storage.AWSSecurityHub{}

	if v, set := os.LookupEnv("NOTIFIER_AWS_ACCOUNT_ID"); set {
		result.AccountId = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_ACCOUNT_ID in env")
	}

	if v, set := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_REGION"); set {
		result.Region = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_REGION in env")
	}

	if v, set := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_ACCESS_KEY_ID"); set {
		result.Credentials = &storage.AWSSecurityHub_Credentials{
			AccessKeyId: v,
		}
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_ACCESS_KEY_ID in env")
	}

	if v, set := os.LookupEnv("NOTIFIER_AWS_SECURITY_HUB_SECRET_ACCESS_KEY"); set {
		result.Credentials.SecretAccessKey = v
	} else {
		return nil, errors.New("missing value for NOTIFIER_AWS_SECURITY_HUB_SECRET_ACCESS_KEY in env")
	}

	return &result, nil
}

func newAlert(state storage.ViolationState) *storage.Alert {
	return &storage.Alert{
		Id: uuid.NewV4().String(),
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
		State:         state,
	}
}

// TestNotifierCreationAndTest exercises very basic functionality. In fact,
// it does not even mutate a security hub instance but rather checks that
// low-level configuration and permissions are correct.
func TestNotifierCreationAndTest(t *testing.T) {
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
	configuration.batchSize = 3
	configuration.uploadTimeout = 1 * time.Second
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
	require.NoError(t, notifier.AlertNotify(ctx, newAlert(storage.ViolationState_ACTIVE)))
	require.NoError(t, notifier.AlertNotify(ctx, newAlert(storage.ViolationState_SNOOZED)))
	require.NoError(t, notifier.AlertNotify(ctx, newAlert(storage.ViolationState_RESOLVED)))

	require.Equal(t, context.DeadlineExceeded, <-errCh)
}
