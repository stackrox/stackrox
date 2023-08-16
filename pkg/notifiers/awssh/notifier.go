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
	"github.com/aws/aws-sdk-go/service/securityhub/securityhubiface"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/logging/structured"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
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
		// From https://docs.aws.amazon.com/securityhub/latest/userguide/finding-update-batchimportfindings.html
		// It is not clear whether they use Decimal or Binary, so we assume 1 KB = 1000 bytes.
		// The maximum finding size is 240 KB, the maximum batch size is 6 MB. We assume each
		// finding will not be larger than 240 KB, giving us a safe batch size of 25.
		maxBatchSize: 25, // 6 MB / 240 KB = 25
		// The throttle rate limit is 10 TPS per account per Region, with burst of 30 TPS.
		// Forgetting the burst, we allow ourselves up to 10 uploads per second.
		minUploadDelay: 100 * time.Millisecond,
		// We set a maximum upload delay to ensure our data stays fresh.
		maxUploadDelay: 15 * time.Second,
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
				log.Debug("ceasing notifier operation", structured.Err(err))
			default:
				log.Error("encountered unexpected error", structured.Err(err))
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

	maxBatchSize   int
	minUploadDelay time.Duration
	maxUploadDelay time.Duration

	canceler func()
}

// notifier is an AlertNotifier implementation.
type notifier struct {
	configuration
	securityHub securityhubiface.SecurityHubAPI
	account     string
	arn         string
	cache       map[string]*storage.Alert
	alertCh     chan *storage.Alert
	initDoneSig concurrency.Signal
	// stoppedSig is owned by the notifier, it is triggered when the `run` method is not executing.
	stoppedSig concurrency.Signal
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
		securityHub:   securityhub.New(awss),
		account:       config.GetAccountId(),
		arn:           product.arnFormatter(config.GetRegion()),
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
		// stoppedSig intentionally omitted - zero value is "already triggered"
	}, nil
}

func (n *notifier) waitForInitDone() {
	n.initDoneSig.Wait()
}

func (n *notifier) sendHeartbeat() {
	now := aws.String(time.Now().UTC().Format(iso8601UTC))
	_, err := n.securityHub.BatchImportFindings(&securityhub.BatchImportFindingsInput{

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
// It will trigger `stoppedSig` once the function exits, signaling its completion.
func (n *notifier) run(ctx context.Context) error {
	if !n.stoppedSig.Reset() {
		// If stoppedSig wasn't triggered before the reset then we were already running.
		n.initDoneSig.Signal()
		return errAlreadyRunning
	}
	defer n.stoppedSig.Signal()

	doneCh := ctx.Done()
	rateLimiter := rate.NewLimiter(rate.Every(n.minUploadDelay), 1)
	uploadTicker := time.NewTicker(n.maxUploadDelay)
	defer uploadTicker.Stop()

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	n.initDoneSig.Signal()

	for {
		select {
		case alert := <-n.alertCh:
			n.processAlert(alert)
			if len(n.cache) >= n.maxBatchSize && rateLimiter.Allow() {
				n.uploadBatch(ctx)
			}
		case <-heartbeatTicker.C:
			n.sendHeartbeat()
		case <-uploadTicker.C:
			if len(n.cache) > 0 && rateLimiter.Allow() {
				n.uploadBatch(ctx)
			}
		case <-doneCh:
			return ctx.Err()
		}
	}
}

func (n *notifier) processAlert(alert *storage.Alert) {
	cached := n.cache[alert.GetId()]
	if cached != nil {
		tsAlert, tsAlertErr := types.TimestampFromProto(alert.GetTime())
		tsCached, tsCachedErr := types.TimestampFromProto(cached.GetTime())

		switch {
		case tsCachedErr != nil || tsCached.Before(tsAlert):
			n.cache[alert.GetId()] = alert
		case tsAlertErr != nil:
			log.Warn("dropping incoming alert with invalid timestamp", structured.Err(tsAlertErr))
		}
	} else {
		n.cache[alert.GetId()] = alert
	}
}

func (n *notifier) uploadBatch(ctx context.Context) {
	if len(n.cache) == 0 {
		log.Debug("no alerts to upload, skipping")
		return
	}

	uiEndpoint := n.ProtoNotifier().GetUiEndpoint()
	batch := &securityhub.BatchImportFindingsInput{}
	alertIds := set.StringSet{}
	for id, alert := range n.cache {
		if len(alertIds) >= n.maxBatchSize {
			break
		}

		finding := mapAlertToFinding(n.account, n.arn, notifiers.AlertLink(uiEndpoint, alert), alert)
		batch.Findings = append(batch.Findings, finding)
		alertIds.Add(id)
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	result, err := n.securityHub.BatchImportFindingsWithContext(ctx, batch)
	if err != nil {
		log.Warn("failed to upload batch", structured.Err(err))
		return
	}

	// Keep alerts that failed to upload in the cache so they'll retry.
	// Note that randomized iteration of map will shuffle failures.
	if result.FailedCount != nil && *result.FailedCount > 0 {
		log.Warn("failed to upload some or all alerts in batch", zap.Any("failures", result.FailedFindings))

		for _, finding := range result.FailedFindings {
			if id := finding.Id; id != nil {
				alertIds.Remove(*id)
			}
		}
	}

	// Remove alerts that successfully uploaded from the cache.
	if len(alertIds) > 0 {
		log.Debug("successfully uploaded some or all alerts in batch", zap.Int("successes", len(alertIds)))
		for id := range alertIds {
			delete(n.cache, id)
		}
	}

	if len(n.cache) >= 5*n.maxBatchSize {
		log.Warn("alert backlog is too large; check for failures above, throttling might need adjusting",
			zap.Int("cacheSize", len(n.cache)))
	}
}

func (n *notifier) Close(_ context.Context) error {
	if n.canceler != nil {
		n.canceler()
	}
	return nil
}

func (n *notifier) ProtoNotifier() *storage.Notifier {
	return n.descriptor
}

// Test checks if:
//   - n is running, i.e., exactly one go routine is executing n.run(...)
//   - AWS SecurityHub is reachable
//
// If either of the checks fails, an error is returned.
func (n *notifier) Test(ctx context.Context) error {
	if n.stoppedSig.IsDone() {
		return errNotRunning
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	_, err := n.securityHub.GetFindingsWithContext(ctx, &securityhub.GetFindingsInput{
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
	_, err = n.securityHub.BatchImportFindings(&securityhub.BatchImportFindingsInput{
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
	if n.stoppedSig.IsDone() {
		return errNotRunning
	}

	if alert.GetImage() != nil {
		return errors.New("AWS SH notifier only supports deployment and resource alerts")
	}

	select {
	case n.alertCh <- alert:
		return nil
	case <-n.stoppedSig.Done():
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
