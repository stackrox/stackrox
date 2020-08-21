// Package awssh provides an AlertNotifier implementation integrating with AWS Security Hub.
package awssh

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	errAlreadyRunning    = errors.New("already running")
	errNotRunning        = errors.New("not running")
	log                  = logging.CurrentModule().Logger()
	defaultConfiguration = configuration{
		upstreamTimeout: 2 * time.Second,
		batchSize:       50,
		uploadTimeout:   15 * time.Second,
	}
	product = struct {
		account string
		arn     string
		name    string
		version string
	}{
		account: "939357552774",
		// arn is the StackRox-provider-specific ARN that we received
		// when registering our integration with AWS Security Hub. If this ARN
		// changes, we need to adjust this constant.
		// arn:aws:securityhub:us-east-2::product/stackrox/kubernetes-security
		// "arn:aws:securityhub:us-east-1:939357552774:product/939357552774/default"
		arn:  "arn:aws:securityhub::939357552774:product/939357552774/default",
		name: "kubernetes-security",
		// TODO(tvoss): Bump to proper version once we default the feature to true.
		version: "0.0.0",
	}
)

func init() {
	if !features.AwsSecurityHubIntegration.Enabled() {
		return
	}

	notifiers.Add("awssh", func(descriptor *storage.Notifier) (notifiers.Notifier, error) {
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

		return notifier, nil
	})
}

func validateNotifierConfiguration(config *storage.AWSSecurityHub) (*storage.AWSSecurityHub, error) {
	if credentials := config.GetCredentials(); credentials != nil {
		accessKeyIDEmpty := credentials.GetAccessKeyId() == ""
		secretAccessKeyEmpty := credentials.GetSecretAccessKey() == ""
		switch {
		case accessKeyIDEmpty && !secretAccessKeyEmpty:
			return nil, errors.New("access key ID must not be empty")
		case !accessKeyIDEmpty && secretAccessKeyEmpty:
			return nil, errors.New("secret access key must not be empty")
		}
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

	awsConfig := aws.NewConfig().WithLogger(aws.LoggerFunc(log.Debug)).WithLogLevel(aws.LogDebug)
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
		cache:         map[string]*storage.Alert{},
		alertCh:       make(chan *storage.Alert),
		initDoneSig:   concurrency.NewSignal(),
	}, nil
}

func (n *notifier) waitForInitDone() {
	n.initDoneSig.Wait()
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
	uploadTimer := time.NewTimer(n.uploadTimeout)
	defer uploadTimer.Stop()

	n.initDoneSig.Signal()

	for {
		select {
		case alert := <-n.alertCh:
			if uploadedBatch := n.processAlert(ctx, alert); uploadedBatch {
				if !uploadTimer.Stop() {
					<-uploadTimer.C
				}
				uploadTimer.Reset(n.uploadTimeout)
			}
		case <-uploadTimer.C:
			// If the upload timer kicks in, we haven't received a new alert for
			// uploadTimeout seconds. We aim to minimize the amount of state
			// kept in memory and drain our internal cache here.
			_ = n.uploadBatchIf(ctx, func(n *notifier) bool {
				return len(n.cache) > 0
			})
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
		input.Findings = append(input.Findings, mapAlertToFinding(alert))
	}

	ctx, cancel := context.WithTimeout(ctx, n.upstreamTimeout)
	defer cancel()

	_, err := n.SecurityHub.BatchImportFindingsWithContext(ctx, input)
	if err != nil {
		log.Warn("failed to upload batch", logging.Err(err))
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
					Value:      aws.String(product.arn),
				},
			},
		},
		MaxResults: aws.Int64(1),
	})
	if err != nil {
		return err
	}

	_, err = n.SecurityHub.BatchImportFindings(&securityhub.BatchImportFindingsInput{
		Findings: []*securityhub.AwsSecurityFinding{
			mapAlertToFinding(&storage.Alert{
				Id: uuid.NewV4().String(),
				Policy: &storage.Policy{
					Id:       uuid.NewV4().String(),
					Name:     "example policy",
					Severity: storage.Severity_HIGH_SEVERITY,
				},
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
				FirstOccurred: types.TimestampNow(),
				Time:          types.TimestampNow(),
				// Mark the state as resolved, thus indicating to security hub that all is good and avoiding raising a false alert.
				State: storage.ViolationState_RESOLVED,
			}),
		},
	})

	return err
}

func (n *notifier) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	if n.stopSig.IsDone() {
		return errNotRunning
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
