// Package awssh provides an AlertNotifier implementation integrating with AWS Security Hub.
package awssh

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	securityhubTypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events/codes"
	"github.com/stackrox/rox/pkg/administration/events/option"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/time/rate"
)

const (
	heartbeatInterval              = 5 * time.Minute
	initialConfigurationMaxTimeout = 5 * time.Minute
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
			cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
			if err != nil {
				utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
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

		// In case STS is not enabled for the integration, expect static configuration to be enabled.
		if !awssh.GetCredentials().GetStsEnabled() {
			if awssh.GetCredentials().GetAccessKeyId() == "" {
				return errors.New("AWS access key ID must not be empty")
			}

			if awssh.GetCredentials().GetSecretAccessKey() == "" {
				return errors.New("AWS secret access key must not be empty")
			}
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

// Returns AccessKeyID and SecretAccessKey from the given awssh notifier
func (c *configuration) getCredentials() (string, string, error) {
	if !env.EncNotifierCreds.BooleanSetting() {
		creds := c.descriptor.GetAwsSecurityHub().GetCredentials()
		return creds.GetAccessKeyId(), creds.GetSecretAccessKey(), nil
	}

	decCredsStr, err := c.cryptoCodec.Decrypt(c.cryptoKey, c.descriptor.GetNotifierSecret())
	if err != nil {
		return "", "", errors.Errorf("Error decrypting notifier secret for notifier '%s'", c.descriptor.GetName())
	}
	creds := &storage.AWSSecurityHub_Credentials{}
	err = creds.UnmarshalVTUnsafe([]byte(decCredsStr))
	if err != nil {
		return "", "", errors.Errorf("Error unmarshalling notifier credentials for notifier '%s'", c.descriptor.GetName())
	}
	return creds.GetAccessKeyId(), creds.GetSecretAccessKey(), nil
}

// notifier is an AlertNotifier implementation.
type notifier struct {
	configuration
	securityHub Client
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

	accessKeyID, secretAccessKey, err := configuration.getCredentials()
	if err != nil {
		return nil, err
	}
	opts := []func(*config.LoadOptions) error{
		config.WithClientLogMode(aws.LogRequestWithBody),
		config.WithHTTPClient(&http.Client{Transport: proxy.RoundTripper()}),
		config.WithRegion(awssh.GetRegion()),
	}
	if !configuration.descriptor.GetAwsSecurityHub().GetCredentials().GetStsEnabled() {
		opts = append(opts,
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
			),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), initialConfigurationMaxTimeout)
	defer cancel()
	awsConfig, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load the aws config")
	}

	return &notifier{
		configuration: configuration,
		securityHub:   securityhub.NewFromConfig(awsConfig),
		account:       awssh.GetAccountId(),
		arn:           product.arnFormatter(awssh.GetRegion()),
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
		// stoppedSig intentionally omitted - zero value is "already triggered"
	}, nil
}

func (n *notifier) waitForInitDone() {
	n.initDoneSig.Wait()
}

func (n *notifier) sendHeartbeat(ctx context.Context) {
	now := aws.String(time.Now().UTC().Format(iso8601UTC))
	input := &securityhub.BatchImportFindingsInput{
		Findings: []securityhubTypes.AwsSecurityFinding{
			{
				SchemaVersion: aws.String(schemaVersion),
				AwsAccountId:  aws.String(n.account),
				ProductArn:    aws.String(n.arn),
				ProductFields: map[string]string{
					"ProviderName":    product.name,
					"ProviderVersion": product.version,
				},
				Description: aws.String("Heartbeat message from StackRox"),
				GeneratorId: aws.String("StackRox"),
				Id:          aws.String("heartbeat-" + *now),
				Title:       aws.String("Heartbeat message from StackRox"),
				Types:       []string{"Heartbeat"},
				CreatedAt:   now,
				UpdatedAt:   now,
				Severity: &securityhubTypes.Severity{
					Normalized: aws.Int32(0),
					Product:    aws.Float64(0),
				},
				Resources: []securityhubTypes.Resource{
					{
						Id:   aws.String("heartbeat-" + *now),
						Type: aws.String(resourceTypeOther),
					},
				},
			},
		},
	}
	if _, err := n.securityHub.BatchImportFindings(ctx, input); err != nil {
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
			n.sendHeartbeat(ctx)
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
		tsAlert, tsAlertErr := protocompat.ConvertTimestampToTimeOrError(alert.GetTime())
		tsCached, tsCachedErr := protocompat.ConvertTimestampToTimeOrError(cached.GetTime())

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

	result, err := n.securityHub.BatchImportFindings(ctx, batch)
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
func (n *notifier) Test(ctx context.Context) *notifiers.NotifierError {
	if n.stoppedSig.IsDone() {
		return notifiers.NewNotifierError(errNotRunning.Error(), nil)
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	_, err := n.securityHub.GetFindings(ctx, &securityhub.GetFindingsInput{
		Filters: &securityhubTypes.AwsSecurityFindingFilters{
			ProductArn: []securityhubTypes.StringFilter{
				{
					Comparison: securityhubTypes.StringFilterComparisonEquals,
					Value:      aws.String(n.arn),
				},
			},
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return notifiers.NewNotifierError("get findings from AWS Security Hub failed", createError("error testing AWS Security Hub integration", err, n.descriptor.GetName()))
	}

	testAlert := storage.Alert_builder{
		Id: uuid.NewV4().String(),
		Policy: storage.Policy_builder{
			Id:          uuid.NewV4().String(),
			Name:        "example policy",
			Severity:    storage.Severity_HIGH_SEVERITY,
			Description: "This finding tests the SecurityHub integration",
		}.Build(),
		Deployment: storage.Alert_Deployment_builder{
			Id:          uuid.NewV4().String(),
			Name:        "example deployment",
			Namespace:   "example namespace",
			ClusterId:   uuid.NewV4().String(),
			ClusterName: "example cluster",
			Containers: []*storage.Alert_Deployment_Container{
				storage.Alert_Deployment_Container_builder{
					Name: "example container",
					Image: storage.ContainerImage_builder{
						Id: uuid.NewV4().String(),
						Name: storage.ImageName_builder{
							FullName: "registry/path/to/image:tag",
						}.Build(),
					}.Build(),
				}.Build(),
			},
		}.Build(),
		FirstOccurred: protocompat.TimestampNow(),
		Time:          protocompat.TimestampNow(),
		// Mark the state as resolved, thus indicating to security hub that all is good and avoiding raising a false alert.
		State: storage.ViolationState_RESOLVED,
	}.Build()
	_, err = n.securityHub.BatchImportFindings(ctx, &securityhub.BatchImportFindingsInput{
		Findings: []securityhubTypes.AwsSecurityFinding{
			mapAlertToFinding(n.account, n.arn, notifiers.AlertLink(n.ProtoNotifier().GetUiEndpoint(), testAlert), testAlert),
		},
	})
	if err != nil {
		return notifiers.NewNotifierError("import test findings to AWS Security Hub failed", createError("error testing AWS Security Hub integration", err, n.descriptor.GetName()))
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
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		if apiErr.ErrorMessage() != "" {
			msg = fmt.Sprintf("%s (code: %s; message: %s)", msg, apiErr.ErrorCode(), apiErr.ErrorMessage())
		} else {
			msg = fmt.Sprintf("%s (code: %s)", msg, apiErr.ErrorCode())
		}
	}
	log.Error("AWS security hub error",
		logging.Err(err),
		logging.NotifierName(notifierName),
	)
	return errors.New(msg)
}
