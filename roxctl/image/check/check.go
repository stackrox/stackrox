package check

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/retry"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/report"
	"google.golang.org/grpc"
)

const (
	jsonFlagName     = "json"
	jsonFailFlagName = "json-fail-on-policy-violations"
)

// Command checks the image against image build lifecycle policies
func Command(cliEnvironment environment.Environment) *cobra.Command {
	imageCheckCmd := &imageCheckCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:  "check",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if err := imageCheckCmd.Construct(nil, c); err != nil {
				return err
			}
			if err := imageCheckCmd.Validate(); err != nil {
				return err
			}
			return imageCheckCmd.CheckImage()
		},
	}

	c.Flags().StringVarP(&imageCheckCmd.image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	pkgUtils.Must(c.MarkFlagRequired("image"))

	c.Flags().BoolVar(&imageCheckCmd.json, jsonFlagName, false, "Output policy results as JSON")
	c.Flags().BoolVar(&imageCheckCmd.failViolationsWithJSON, jsonFailFlagName, true,
		"Whether policy violations should cause the command to exit non-zero in JSON output mode too. "+
			"This flag only has effect when --json is also specified.")
	c.Flags().IntVarP(&imageCheckCmd.retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds.")
	c.Flags().IntVarP(&imageCheckCmd.retryCount, "retries", "r", 0, "number of retries before exiting as error.")
	c.Flags().BoolVar(&imageCheckCmd.sendNotifications, "send-notifications", false,
		"whether to send notifications for violations (notifications will be sent to the notifiers "+
			"configured in each violated policy).")
	c.Flags().StringSliceVarP(&imageCheckCmd.policyCategories, "categories", "c", nil, "optional comma separated list of policy categories to run.  Defaults to all policy categories.")
	c.Flags().BoolVar(&imageCheckCmd.printAllViolations, "print-all-violations", false, "whether to print all violations per alert or truncate violations for readability")
	return c
}

// imageCheckCommand holds all configurations and metadata to execute an image check
type imageCheckCommand struct {
	image                  string
	json                   bool
	failViolationsWithJSON bool
	retryDelay             int
	retryCount             int
	sendNotifications      bool
	policyCategories       []string
	printAllViolations     bool
	timeout                time.Duration
	env                    environment.Environment
	centralDetectionSvc    centralDetectionService
}

// centralDetectionService abstracts away the gRPC call to the detection service within central.
// this is especially useful for testing
type centralDetectionService interface {
	DetectBuildTime(ctx context.Context, in *v1.BuildDetectionRequest, opts ...grpc.CallOption) (*v1.BuildDetectionResponse, error)
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (i *imageCheckCommand) Construct(args []string, cmd *cobra.Command) error {
	i.timeout = flags.Timeout(cmd)
	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (i *imageCheckCommand) Validate() error {
	if i.failViolationsWithJSON && !i.json {
		fmt.Fprintf(i.env.InputOutput().ErrOut, "Note: --%s has no effect when --%s is not specified.\n", jsonFailFlagName, jsonFlagName)
	}
	return nil
}

// CheckImage will execute the image check with retry functionality
func (i *imageCheckCommand) CheckImage() error {
	err := retry.WithRetry(func() error {
		return i.checkImage()
	},
		retry.Tries(i.retryCount+1),
		retry.OnFailedAttempts(func(err error) {
			fmt.Fprintf(i.env.InputOutput().ErrOut, "Checking image failed: %v. Retrying after %v seconds\n", err, i.retryDelay)
			time.Sleep(time.Duration(i.retryDelay) * time.Second)
		}))
	if err != nil {
		return err
	}
	return nil
}

func (i *imageCheckCommand) checkImage() error {
	// Get the violated policies for the input data.
	req, err := buildRequest(i.image, i.sendNotifications, i.policyCategories)
	if err != nil {
		return err
	}
	alerts, err := i.getAlerts(req)
	if err != nil {
		return err
	}

	return i.reportCheckResults(alerts)
}

func (i *imageCheckCommand) reportCheckResults(alerts []*storage.Alert) error {
	// If json mode was given, print results (as json) and either immediately return or check policy,
	// depending on a flag.
	if i.json {
		err := report.JSON(i.env.InputOutput().Out, alerts)
		if err != nil {
			return err
		}
		if i.failViolationsWithJSON {
			return checkPolicyFailures(alerts)
		}
		return nil
	}

	// Print results in human readable mode.
	if err := report.PrettyWithResourceName(i.env.InputOutput().Out, alerts, storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT, "Image", i.image, i.printAllViolations); err != nil {
		return err
	}

	return checkPolicyFailures(alerts)
}

func (i *imageCheckCommand) getAlerts(req *v1.BuildDetectionRequest) ([]*storage.Alert, error) {
	conn, err := i.env.GRPCConnection()
	if err != nil {
		return nil, err
	}

	defer pkgUtils.IgnoreError(conn.Close)
	svc := v1.NewDetectionServiceClient(conn)

	i.centralDetectionSvc = svc

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), i.timeout)
	defer cancel()

	response, err := i.centralDetectionSvc.DetectBuildTime(ctx, req)
	if err != nil {
		return nil, err
	}

	return response.GetAlerts(), err
}

func checkPolicyFailures(alerts []*storage.Alert) error {
	// Check if any of the violated policies have an enforcement action that
	// fails the CI build.
	for _, alert := range alerts {
		if report.EnforcementFailedBuild(storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT)(alert.GetPolicy()) {
			return errors.New("Violated a policy with CI enforcement set")
		}
	}
	return nil
}

// Use inputs to generate an image name for request.
func buildRequest(image string, sendNotifications bool, policyCategories []string) (*v1.BuildDetectionRequest, error) {
	img, err := utils.GenerateImageFromString(image)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse image '%s'", image)
	}
	return &v1.BuildDetectionRequest{
		Resource:          &v1.BuildDetectionRequest_Image{Image: img},
		SendNotifications: sendNotifications,
		PolicyCategories:  policyCategories,
	}, nil
}
