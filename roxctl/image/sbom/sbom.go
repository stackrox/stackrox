package sbom

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/apiparams"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	imageSBOMAPIPath = "/api/v1/images/sbom"
)

// Command sends a request to Central to generate an SBOM from an image scan.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	imageSBOMCmd := &imageSBOMCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "sbom",
		Short: "Generate an SPDX 2.3 SBOM from an image scan.",
		Long:  "Generate an SPDX 2.3 SBOM from an image scan. You must have write permissions for the `Image` resource.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := imageSBOMCmd.construct(c); err != nil {
				return err
			}

			return imageSBOMCmd.GenerateSBOM()
		}),
	}

	c.Flags().StringVarP(&imageSBOMCmd.image, "image", "i", "", "Image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVarP(&imageSBOMCmd.force, "force", "f", false, "Bypass Central's cache for the image and force a new pull from the Scanner")
	// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
	// c.Flags().StringVar(&imageSBOMCmd.cluster, "cluster", "", "Cluster name or ID to delegate image scan to")
	c.Flags().IntVarP(&imageSBOMCmd.retryDelay, "retry-delay", "d", 3, "Set time to wait between retries in seconds")
	c.Flags().IntVarP(&imageSBOMCmd.retryCount, "retries", "r", 3, "Number of retries before exiting as error")

	utils.Must(c.MarkFlagRequired("image"))
	return c
}

// imageSBOMCommand holds all configurations and metadata to generate an SBOM.
type imageSBOMCommand struct {
	image string
	force bool
	// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
	// cluster    string
	retryDelay int
	retryCount int

	env    environment.Environment
	client common.RoxctlHTTPClient

	// The HTTP request body to send to Central.
	requestBody []byte
}

// construct ensures all flag values are valid and HTTP client/request built.
func (i *imageSBOMCommand) construct(cobraCmd *cobra.Command) error {
	var err error

	// Validate the image reference.
	if err := imageUtils.IsValidImageString(i.image); err != nil {
		return common.ErrInvalidCommandOption.CausedBy(errors.Wrap(err, "image"))
	}

	// Create the HTTP client.
	i.client, err = i.env.HTTPClient(
		flags.Timeout(cobraCmd),
		// Disable exponential backoff so that roxctl does not appear stuck.
		common.WithRetryExponentialBackoff(false),
		// Ensure error response is made available for troubleshooting failures.
		common.WithReturnErrorResponseBody(true),
		common.WithRetryDelay(time.Duration(i.retryDelay)*time.Second),
		common.WithRetryCount(i.retryCount),
	)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}

	// Create the request.
	req := apiparams.SBOMRequestBody{
		// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
		// Cluster:   i.cluster,
		ImageName: i.image,
		Force:     i.force,
	}

	i.requestBody, err = json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "creating request body")
	}

	return nil
}

func (i *imageSBOMCommand) GenerateSBOM() error {
	// Send HTTP request and verify response status code.
	resp, err := i.client.DoReqAndVerifyStatusCode(
		imageSBOMAPIPath,
		http.MethodPost,
		http.StatusOK,
		bytes.NewReader(i.requestBody),
	)
	if err != nil {
		return errors.Wrap(err, "generating SBOM")
	}
	defer utils.IgnoreError(resp.Body.Close)

	// Central returns a 200 response with Content-Type text/html for any unimplemented '/api/*' endpoints,
	// catch this and return an error.
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return errors.Errorf("unexpected Content-Type %q from Central, confirm Central version supports SBOM generation", contentType)
	}

	// Output the raw SBOM.
	_, err = io.Copy(i.env.InputOutput().Out(), resp.Body)
	if err != nil {
		return errors.Wrap(err, "writing response")
	}

	return nil
}
