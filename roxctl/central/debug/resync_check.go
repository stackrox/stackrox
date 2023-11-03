package debug

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc"
)

const (
	defaultResyncCheckTimeout = 300 * time.Second
)

// resyncCheckCommand will outputs alert data before and after reassessing ACS policies.
func resyncCheckCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputDir string
	var waitFor time.Duration

	c := &cobra.Command{
		Use:   "resync-check",
		Short: "Check alerts before and after reassessing policies",
		Long:  "Check alerts before and after reassessing policies. This should only be used for testing when Secured Clusters have ROX_RESYNC_DISABLED=true",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cmd, err := commandWithConnection(cliEnvironment, waitFor, flags.Timeout(c), outputDir)
			if err != nil {
				return err
			}
			before, after, err := cmd.run()
			if err != nil {
				return err
			}

			return cmd.storeFiles(before, after)
		}),
	}
	flags.AddTimeoutWithDefault(c, defaultResyncCheckTimeout)
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "resync-check-output", "output directory in which to store bundle")
	c.PersistentFlags().DurationVar(&waitFor, "wait-for", time.Minute, "how long to wait between before and after alert check")

	return c
}

type resyncCheckCmd struct {
	env       environment.Environment
	conn      *grpc.ClientConn
	waitFor   time.Duration
	timeout   time.Duration
	outputDir string
}

func commandWithConnection(env environment.Environment, waitFor, timeout time.Duration, outputDir string) (*resyncCheckCmd, error) {
	conn, err := env.GRPCConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to central")
	}

	return &resyncCheckCmd{
		env:       env,
		conn:      conn,
		waitFor:   waitFor,
		timeout:   timeout,
		outputDir: outputDir,
	}, nil
}

func (c *resyncCheckCmd) run() ([]*storage.ListAlert, []*storage.ListAlert, error) {
	c.env.Logger().InfofLn("Running re-sync check")

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	svc := v1.NewAlertServiceClient(c.conn)

	alertsBefore, err := c.fetchAlerts(ctx, svc)
	if err != nil {
		return nil, nil, err
	}
	c.env.Logger().InfofLn("Found %d alerts before reassessing", len(alertsBefore))

	if err := c.reassessPolicies(ctx); err != nil {
		return nil, nil, err
	}

	c.env.Logger().InfofLn("Waiting for %s while reassess policies produces alerts", c.waitFor.String())

	// Roxctl needs to wait here to realistically get the delta between the violation state before and after reassessing
	// policies. If the timer is too short, there's a higher change that there will be no delta, because the Sensors
	// still haven't fully reprocessed the deployments.
	waitForC := time.After(c.waitFor)
	select {
	case <-waitForC:
		break
	case <-ctx.Done():
		return nil, nil, errox.InvalidArgs.Newf("context finished before second request could be made: `wait-for` value might be too high: %s", c.waitFor.String())
	}

	alertsAfter, err := c.fetchAlerts(ctx, svc)
	if err != nil {
		return nil, nil, err
	}
	c.env.Logger().InfofLn("Found %d alerts after reassessing", len(alertsAfter))

	c.assessDelta(alertsBefore, alertsAfter)

	return alertsBefore, alertsAfter, nil
}

func (c *resyncCheckCmd) fetchAlerts(ctx context.Context, svc v1.AlertServiceClient) ([]*storage.ListAlert, error) {
	response, err := svc.ListAlerts(ctx, &v1.ListAlertsRequest{
		Pagination: &v1.Pagination{
			Limit: math.MaxInt32,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not make ListAlerts request to gRPC server")
	}
	return response.GetAlerts(), nil
}

func (c *resyncCheckCmd) storeFiles(alertsBefore, alertsAfter []*storage.ListAlert) error {
	if err := os.Mkdir(c.outputDir, os.ModePerm); err != nil && !os.IsExist(err) {
		c.env.Logger().WarnfLn("Failed to create directory %s: %s", c.outputDir, err)
	}

	if err := c.storeFile("alerts-before.json", alertsBefore); err != nil {
		return err
	}

	if err := c.storeFile("alerts-after.json", alertsAfter); err != nil {
		return err
	}

	return nil
}

func (c *resyncCheckCmd) storeFile(fileName string, alerts []*storage.ListAlert) error {
	fullPath := path.Join(c.outputDir, fileName)
	c.env.Logger().InfofLn("Storing alerts %s", fullPath)

	data, err := json.Marshal(alerts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ListAlerts as JSON")
	}

	return os.WriteFile(fullPath, data, 0o644)
}

func (c *resyncCheckCmd) assessDelta(before, after []*storage.ListAlert) {
	if len(before) != len(after) {
		c.env.Logger().WarnfLn("Number of alerts differ in before and after! Reach out to Red Hat ACS support and provide output files generated by this command.")
		return
	}

	sort.SliceStable(before, func(i, j int) bool {
		return before[i].Id > before[j].Id
	})

	sort.SliceStable(after, func(i, j int) bool {
		return after[i].Id > after[j].Id
	})

	if !protoutils.SlicesEqual(before, after) {
		c.env.Logger().WarnfLn("Alerts content differ in before and after! Reach out to Red Hat ACS support and provide output files generated by this command.")
	}
}

func (c *resyncCheckCmd) reassessPolicies(ctx context.Context) error {
	policySvc := v1.NewPolicyServiceClient(c.conn)
	_, err := policySvc.ReassessPolicies(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "couldn't reassess policies")
	}
	return nil
}
