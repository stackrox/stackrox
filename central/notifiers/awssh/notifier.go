// Package awssh provides an AlertNotifier implementation integrating with AWS Security Hub.
package awssh

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap"
)

const (
	heartbeatInterval = 5 * time.Minute
)

var (
	errAlreadyRunning    = errors.New("already running")
	errNotRunning        = errors.New("not running")
	log                  = logging.CurrentModule().Logger()
	defaultConfiguration = configuration{
		upstreamTimeout: 2 * time.Second,
		batchSize:       5,
		uploadTimeout:   15 * time.Second,
	}
	product = struct {
		arnFormatter func(string) string
		name         string
		version      string
	}{
		// arn is the StackRox-provider-specific ARN that we received
		// when registering our integration with AWS Security Hub. If this ARN
		// changes, we need to adjust this constant.
		arnFormatter: func(region string) string {
			return fmt.Sprintf("arn:aws:securityhub:%s::product/stackrox/kubernetes-security", region)
		},
		name:    "kubernetes-security",
		version: "1.0.0",
	}
)

func init() {
	notifiers.Add("awsSecurityHub", func(descriptor *storage.Notifier) (notifiers.Notifier, error) {
		ctx, cancel := context.WithCancel(context.Background())
		configuration := defaultConfiguration
		configuration.descriptor = descriptor
		configuration.canceler = cancel

		notifier, err := newNotifier(configuration)
		if err != nil {
			return nil, err
		}

		go func() {
			switch err := notifier.run(ctx); err {
			case nil, context.Canceled, context.DeadlineExceeded:
				log.Debug("ceasing notifier operation", logging.Err(err))
			default:
				log.Error("encountered unexpected error", logging.Err(err))
			}
		}()

		notifier.waitForInitDone()

		return notifier, nil
	})
}

func validateNotifierConfiguration(config *storage.AWSSecurityHub) (*storage.AWSSecurityHub, error) {
	if config.GetRegion() == "" {
		return nil, errors.New("AWS region must not be empty")
	}

	if config.GetAccountId() == "" {
		return nil, errors.New("AWS account ID must not be empty")
	}

	if config.GetCredentials() == nil {
		return nil, errors.New("AWS credentials must not be empty")
	}

	if config.GetCredentials().GetAccessKeyId() == "" {
		return nil, errors.New("AWS access key ID must not be empty")
	}

	if config.GetCredentials().GetSecretAccessKey() == "" {
		return nil, errors.New("AWS secret access key must not be empty")
	}

	return config, nil
}

type configuration struct {
	descriptor      *storage.Notifier
	upstreamTimeout time.Duration
	batchSize       int
	uploadTimeout   time.Duration
	canceler        func()
}

// notifier is an AlertNotifier implementation.
type notifier struct {
	configuration
	*securityhub.SecurityHub
	account     string
	arn         string
	cache       map[string]*storage.Alert
	alertCh     chan *storage.Alert
	stopSig     concurrency.Signal
	initDoneSig concurrency.Signal
}

func newNotifier(configuration configuration) (*notifier, error) {
	config, err := validateNotifierConfiguration(configuration.descriptor.GetAwsSecurityHub())
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate config for AWS SecurityHub")
	}

	awsConfig := aws.NewConfig().WithLogger(aws.LoggerFunc(log.Debug)).WithLogLevel(aws.LogDebugWithHTTPBody)
	if region := config.GetRegion(); region != "" {
		awsConfig = awsConfig.WithRegion(config.GetRegion())
	}
	if creds := config.GetCredentials(); creds != nil {
		awsConfig = awsConfig.WithCredentials(credentials.NewStaticCredentials(
			creds.GetAccessKeyId(),
			creds.GetSecretAccessKey(),
			"",
		))
	}

	awss, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}

	return &notifier{
		configuration: configuration,
		SecurityHub:   securityhub.New(awss),
		account:       config.GetAccountId(),
		arn:           product.arnFormatter(config.GetRegion()),
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
	}, nil
}

func (n *notifier) waitForInitDone() {
	n.initDoneSig.Wait()
}

func (n *notifier) sendHeartbeat() {
	now := aws.String(time.Now().UTC().Format(iso8601UTC))
	_, err := n.SecurityHub.BatchImportFindings(&securityhub.BatchImportFindingsInput{

		Findings: []*securityhub.AwsSecurityFinding{
			{
				SchemaVersion: aws.String(schemaVersion),
				AwsAccountId:  aws.String(n.account),
				ProductArn:    aws.String(n.arn),
				ProductFields: map[string]*string{
					"ProviderName":    aws.String(product.name),
					"ProviderVersion": aws.String(product.version),
				},
				Description: aws.String("Heartbeat message from StackRox"),
				GeneratorId: aws.String("StackRox"),
				Id:          aws.String("heartbeat-" + *now),
				Title:       aws.String("Heartbeat message from StackRox"),
				Types: []*string{
					aws.String("Heartbeat"),
				},
				CreatedAt: now,
				UpdatedAt: now,
				Severity: &securityhub.Severity{
					Normalized: aws.Int64(0),
					Product:    aws.Float64(0),
				},
				Resources: []*securityhub.Resource{
					{
						Id:   aws.String("heartbeat-" + *now),
						Type: aws.String(resourceTypeOther),
					},
				},
			},
		},
	})
	if err != nil {
		log.Errorf("unable to send heartbeat to AWS SecurityHub: %v", err)
	}
}

// run executes n's event processing loop until either an error occurs or ctx is marked as done.
// If syncer is not nil, run writes to syncer when initialization is done (or an error occured).
func (n *notifier) run(ctx context.Context) error {
	if !n.stopSig.Reset() {
		n.initDoneSig.Signal()
		return errAlreadyRunning
	}
	defer func() {
		n.stopSig.Signal()
	}()

	doneCh := ctx.Done()
	lastUpload := time.Time{}
	uploadTicker := time.NewTicker(n.uploadTimeout)
	defer uploadTicker.Stop()

	heartbeatTicker := time.NewTicker(heartbeatInterval)

	n.initDoneSig.Signal()

	for {
		select {
		case alert := <-n.alertCh:
			if n.processAlert(ctx, alert) {
				lastUpload = time.Now()
			}
		case <-heartbeatTicker.C:
			n.sendHeartbeat()
		case <-uploadTicker.C:
			// If the upload timer kicks in, we haven't received a new alert for
			// uploadTimeout seconds. We aim to minimize the amount of state
			// kept in memory and drain our internal cache here.
			if time.Since(lastUpload) > n.uploadTimeout && n.uploadBatchIf(ctx, func(n *notifier) bool {
				return len(n.cache) > 0
			}) {
				lastUpload = time.Now()
			}
		case <-doneCh:
			return ctx.Err()
		}
	}
}

func (n *notifier) processAlert(ctx context.Context, alert *storage.Alert) bool {
	cached := n.cache[alert.GetId()]
	if cached != nil {
		tsAlert, tsAlertErr := types.TimestampFromProto(alert.GetTime())
		tsCached, tsCachedErr := types.TimestampFromProto(cached.GetTime())

		switch {
		case tsCachedErr != nil || tsCached.Before(tsAlert):
			n.cache[alert.GetId()] = alert
		case tsAlertErr != nil:
			log.Warn("dropping incoming alert with invalid timestamp", logging.Err(tsAlertErr))
		}
	} else {
		n.cache[alert.GetId()] = alert
	}

	return n.uploadBatchIf(ctx, func(n *notifier) bool {
		return len(n.cache) >= n.batchSize
	})
}

func (n *notifier) uploadBatchIf(ctx context.Context, predicate func(*notifier) bool) bool {
	if !predicate(n) {
		log.Debug("skipping upload")
		return false
	}

	input := &securityhub.BatchImportFindingsInput{}

	for _, alert := range n.cache {
		input.Findings = append(input.Findings, mapAlertToFinding(n.account, n.arn, notifiers.AlertLink(n.ProtoNotifier().GetUiEndpoint(), alert), alert))
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	result, err := n.SecurityHub.BatchImportFindingsWithContext(ctx, input)
	if err != nil {
		log.Warn("failed to upload batch", logging.Err(err))
		return false
	}

	if result.FailedCount != nil && *result.FailedCount > 0 {
		cache := make(map[string]*storage.Alert)
		for _, finding := range result.FailedFindings {
			if id := finding.Id; id != nil {
				if entry := n.cache[*id]; entry != nil {
					cache[*id] = entry
				}
			}
		}
		log.Warn("failed to upload batch", zap.Any("failures", result.FailedFindings))
		n.cache = cache

		return false
	}

	log.Debug("successfully uploaded batch")
	n.cache = make(map[string]*storage.Alert)

	return true
}

func (n *notifier) Close(ctx context.Context) error {
	if n.canceler != nil {
		n.canceler()
	}
	return nil
}

func (n *notifier) ProtoNotifier() *storage.Notifier {
	return n.descriptor
}

// Test checks if:
//   * n is running, i.e., exactly one go routine is executing n.run(...)
//   * AWS SecurityHub is reachable
// If either of the checks fails, an error is returned.
func (n *notifier) Test(ctx context.Context) error {
	if n.stopSig.IsDone() {
		return errNotRunning
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	_, err := n.SecurityHub.GetFindingsWithContext(ctx, &securityhub.GetFindingsInput{
		Filters: &securityhub.AwsSecurityFindingFilters{
			ProductArn: []*securityhub.StringFilter{
				{
					Comparison: aws.String(securityhub.StringFilterComparisonEquals),
					Value:      aws.String(n.arn),
				},
			},
		},
		MaxResults: aws.Int64(1),
	})
	if err != nil {
		return createError("error testing AWS Security Hub integration", err)
	}

	testAlert := &storage.Alert{
		Id: uuid.NewV4().String(),
		Policy: &storage.Policy{
			Id:          uuid.NewV4().String(),
			Name:        "example policy",
			Severity:    storage.Severity_HIGH_SEVERITY,
			Description: "This finding tests the SecurityHub integration",
		},
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
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
		}},
		FirstOccurred: types.TimestampNow(),
		Time:          types.TimestampNow(),
		// Mark the state as resolved, thus indicating to security hub that all is good and avoiding raising a false alert.
		State: storage.ViolationState_RESOLVED,
	}
	_, err = n.SecurityHub.BatchImportFindings(&securityhub.BatchImportFindingsInput{
		Findings: []*securityhub.AwsSecurityFinding{
			mapAlertToFinding(n.account, n.arn, notifiers.AlertLink(n.ProtoNotifier().GetUiEndpoint(), testAlert), testAlert),
		},
	})

	if err != nil {
		return createError("error testing AWS Security Hub integration", err)
	}
	return nil

}

func (n *notifier) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	if n.stopSig.IsDone() {
		return errNotRunning
	}

	if alert.GetDeployment() == nil {
		return errors.New("AWS SH notifier only supports deployment alerts")
	}

	select {
	case n.alertCh <- alert:
		return nil
	case <-n.stopSig.Done():
		return errNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *notifier) AckAlert(_ context.Context, _ *storage.Alert) error {
	return nil
}

func (n *notifier) ResolveAlert(ctx context.Context, alert *storage.Alert) error {
	return n.AlertNotify(ctx, alert)
}

func createError(msg string, err error) error {
	if awsErr, _ := err.(awserr.Error); awsErr != nil {
		if awsErr.Message() != "" {
			msg = fmt.Sprintf("%s (code: %s; message: %s)", msg, awsErr.Code(), awsErr.Message())
		} else {
			msg = fmt.Sprintf("%s (code: %s)", msg, awsErr.Code())
		}
	}
	log.Errorf("AWS Security hub error: %v", err)
	return errors.New(msg)
}
