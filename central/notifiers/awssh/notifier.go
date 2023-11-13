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
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/time/rate"
)

const (
	heartbeatInterval = 5 * time.Minute
)

var (
	errAlreadyRunning    = errors.New("already running")
	errNotRunning        = errors.New("not running")
	log                  = logging.LoggerForModule(option.EnableAdministrationEvents())
	defaultConfiguration = configuration{
		upstreamTimeout: env.AWSSHUploadTimeout.DurationSetting(),
		// From https://docs.aws.amazon.com/securityhub/latest/userguide/finding-update-batchimportfindings.html
		// It is not clear whether they use Decimal or Binary, so we assume 1 KB = 1000 bytes.
		// The maximum finding size is 240 KB, the maximum batch size is 6 MB. We assume each
		// finding will not be larger than 240 KB, giving us a safe batch size of 25.
		maxBatchSize: 25, // 6 MB / 240 KB = 25
		// The throttle rate limit is 10 TPS per account per Region, with burst of 30 TPS.
		// Forgetting the burst, we allow ourselves up to 10 uploads per second.
		minUploadDelay: 100 * time.Millisecond,
		// We set a maximum upload delay to ensure our data stays fresh.
		maxUploadDelay: env.AWSSHUploadInterval.DurationSetting(),
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
	notifiers.Add(notifiers.AWSSecurityHubType, func(descriptor *storage.Notifier) (notifiers.Notifier, error) {
		ctx, cancel := context.WithCancel(context.Background())
		configuration := defaultConfiguration
		configuration.descriptor = descriptor
		configuration.canceler = cancel

		cryptoKey := ""
		var err error
		if env.EncNotifierCreds.BooleanSetting() {
			cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
			if err != nil {
				utils.CrashOnError(err)
			}
		}
		configuration.cryptoKey = cryptoKey
		configuration.cryptoCodec = cryptocodec.Singleton()

		notifier, err := newNotifier(configuration)
		if err != nil {
			return nil, err
		}

		go func() {
			switch err := notifier.run(ctx); err {
			case nil, context.Canceled, context.DeadlineExceeded:
				log.Debug("ceasing notifier operation", logging.Err(err))
			default:
				log.Errorw("encountered unexpected error",
					logging.Err(err),
					logging.NotifierName(notifier.descriptor.GetName()),
					logging.ErrCode(codes.AWSSHGeneric))
			}
		}()

		notifier.waitForInitDone()

		return notifier, nil
	})
}

// Validate AWSSecurityHub notifier
func Validate(awssh *storage.AWSSecurityHub, validateSecret bool) error {
	if awssh == nil {
		return errors.New("AWSSecurityHub configuration is required")
	}

	if awssh.GetRegion() == "" {
		return errors.New("AWS region must not be empty")
	}

	if awssh.GetAccountId() == "" {
		return errors.New("AWS account ID must not be empty")
	}

	if validateSecret {
		if awssh.GetCredentials() == nil {
			return errors.New("AWS credentials must not be empty")
		}

		if awssh.GetCredentials().GetAccessKeyId() == "" {
			return errors.New("AWS access key ID must not be empty")
		}

		if awssh.GetCredentials().GetSecretAccessKey() == "" {
			return errors.New("AWS secret access key must not be empty")
		}
	}

	return nil
}

type configuration struct {
	descriptor      *storage.Notifier
	upstreamTimeout time.Duration

	maxBatchSize   int
	minUploadDelay time.Duration
	maxUploadDelay time.Duration

	cryptoKey   string
	cryptoCodec cryptocodec.CryptoCodec

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
	awssh := configuration.descriptor.GetAwsSecurityHub()
	err := Validate(awssh, !env.EncNotifierCreds.BooleanSetting())
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate config for AWS SecurityHub")
	}

	awsConfig := aws.NewConfig().WithLogger(aws.LoggerFunc(log.Debug)).WithLogLevel(aws.LogDebugWithHTTPBody)
	if region := awssh.GetRegion(); region != "" {
		awsConfig = awsConfig.WithRegion(awssh.GetRegion())
	}

	creds, err := getCredentials(configuration)
	if err != nil {
		return nil, err
	}
	if creds != nil {
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
		account:       awssh.GetAccountId(),
		arn:           product.arnFormatter(awssh.GetRegion()),
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
		// stoppedSig intentionally omitted - zero value is "already triggered"
	}, nil
}

func getCredentials(config configuration) (*storage.AWSSecurityHub_Credentials, error) {
	if !env.EncNotifierCreds.BooleanSetting() {
		return config.descriptor.GetAwsSecurityHub().GetCredentials(), nil
	}

	if config.descriptor.GetNotifierSecret() == "" {
		return nil, errors.Errorf("encrypted notifier credentials for notifier '%s' empty", config.descriptor.GetName())
	}

	decCredsStr, err := config.cryptoCodec.Decrypt(config.cryptoKey, config.descriptor.GetNotifierSecret())
	if err != nil {
		return nil, errors.Errorf("Error decrypting notifier secret for notifier '%s'", config.descriptor.GetName())
	}
	creds := &storage.AWSSecurityHub_Credentials{}
	err = creds.Unmarshal([]byte(decCredsStr))
	if err != nil {
		return nil, errors.Errorf("Error unmarshalling notifier credentials for notifier '%s'", config.descriptor.GetName())
	}
	return creds, err
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
		log.Errorw("unable to send heartbeat to AWS SecurityHub",
			logging.Err(err), logging.NotifierName(n.descriptor.GetName()),
			logging.ErrCode(codes.AWSSHHeartBeat))
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
			log.Warnw("dropping incoming alert with invalid timestamp",
				logging.Err(tsAlertErr),
				logging.NotifierName(n.descriptor.GetName()),
				logging.ErrCode(codes.AWSSHInvalidTimestamp),
				logging.AlertID(alert.GetId()))
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
		log.Warnw("failed to upload batch",
			logging.Err(err),
			logging.NotifierName(n.descriptor.GetName()),
			logging.ErrCode(codes.AWSSHBatchUpload))
		return
	}

	// Keep alerts that failed to upload in the cache so they'll retry.
	// Note that randomized iteration of map will shuffle failures.
	if result.FailedCount != nil && *result.FailedCount > 0 {
		log.Warnw("failed to upload some or all alerts in batch",
			logging.Any("failures", result.FailedFindings),
			logging.ErrCode(codes.AWSSHBatchUpload),
			logging.NotifierName(n.descriptor.GetName()))

		for _, finding := range result.FailedFindings {
			if id := finding.Id; id != nil {
				alertIds.Remove(*id)
			}
		}
	}

	// Remove alerts that successfully uploaded from the cache.
	if len(alertIds) > 0 {
		log.Debug("successfully uploaded some or all alerts in batch",
			logging.Int("successes", len(alertIds)))
		for id := range alertIds {
			delete(n.cache, id)
		}
	}

	if len(n.cache) >= 5*n.maxBatchSize {
		log.Warnw("alert backlog is too large; throttling might need adjusting",
			logging.Int("cacheSize", len(n.cache)),
			logging.ErrCode(codes.AWSSHCacheExhausted),
			logging.NotifierName(n.descriptor.GetName()))
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
		return createError("error testing AWS Security Hub integration", err, n.descriptor.GetName())
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
		return createError("error testing AWS Security Hub integration", err, n.descriptor.GetName())
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

func createError(msg string, err error, notifierName string) error {
	if awsErr, _ := err.(awserr.Error); awsErr != nil {
		if awsErr.Message() != "" {
			msg = fmt.Sprintf("%s (code: %s; message: %s)", msg, awsErr.Code(), awsErr.Message())
		} else {
			msg = fmt.Sprintf("%s (code: %s)", msg, awsErr.Code())
		}
	}
	log.Error("AWS Security hub error",
		logging.Err(err),
		logging.NotifierName(notifierName))
	return errors.New(msg)
}
