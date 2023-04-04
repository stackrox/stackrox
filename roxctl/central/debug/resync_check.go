package debug

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	resyncCheckTimeout = 300 * time.Second
)

// resyncCheckCommand command to debug re-sync-less feature
func resyncCheckCommand(cliEnvironment environment.Environment) *cobra.Command {
	var outputDir string
	var clusters []string

	c := &cobra.Command{
		Use:   "resync-check",
		Short: "TODO",
		Long:  "TODO",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cliEnvironment.Logger().InfofLn("Downloading alerts from central...")

			timeout := flags.Timeout(c)
			alertsPath := "/v1/alerts"
			reassessPath := "/v1/policies/reassess"

			err := getFile(alertsPath, timeout, http.MethodGet, "alerts-before.json")
			if isTimeoutError(err) {
				cliEnvironment.Logger().ErrfLn("Timeout has been reached while retrieving the alerts")
				return nil
			} else if err != nil {
				return err
			}

			cliEnvironment.Logger().InfofLn("Triggering policy reassess...")
			var body []byte
			resp, err := common.DoHTTPRequestAndCheck200(reassessPath, timeout, http.MethodPost, bytes.NewBuffer(body), environment.CLIEnvironment().Logger())
			if isTimeoutError(err) {
				cliEnvironment.Logger().ErrfLn("Timeout has been reached while requesting policy reassess")
				return nil
			} else if err != nil {
				return err
			}
			utils.IgnoreError(resp.Body.Close)

			cliEnvironment.Logger().InfofLn("Sleeping one minute...")
			time.Sleep(time.Minute)

			cliEnvironment.Logger().InfofLn("Downloading alerts from central...")
			err = getFile(alertsPath, timeout, http.MethodGet, "alerts-after.json")
			if isTimeoutError(err) {
				cliEnvironment.Logger().ErrfLn("Timeout has been reached while retrieving the alerts")
			}
			return err
		}),
	}
	flags.AddTimeoutWithDefault(c, diagnosticBundleDownloadTimeout)
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "output directory in which to store bundle")
	c.PersistentFlags().StringSliceVar(&clusters, "clusters", nil, "comma separated list of sensor clusters from which logs should be collected")

	return c
}

func getFile(path string, timeout time.Duration, method, fileName string) error {
	var body []byte
	resp, err := common.DoHTTPRequestAndCheck200(path, timeout, method, bytes.NewBuffer(body), environment.CLIEnvironment().Logger())
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(out.Close)
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}
