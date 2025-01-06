package sbom

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		Short: "Generate an SBOM for the specified image.",
		Long:  "Generate an SBOM for the specified image from an image scan. Optionally, force a rescan of the image. You must have write permissions for the `Image` resource.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := imageSBOMCmd.construct(c); err != nil {
				return err
			}

			return imageSBOMCmd.GenerateSBOM()
		}),
	}

	c.Flags().StringVarP(&imageSBOMCmd.image, "image", "i", "", "Image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVarP(&imageSBOMCmd.force, "force", "f", false, "The --force flag ignores Central's cache for the scan and forces a fresh re-pull from Scanner")
	c.Flags().StringVar(&imageSBOMCmd.cluster, "cluster", "", "Cluster name or ID to delegate image scan to")

	utils.Must(c.MarkFlagRequired("image"))
	return c
}

// imageSBOMCommand holds all configurations and metadata to generate an SBOM.
type imageSBOMCommand struct {
	image string
	force bool

	timeout time.Duration
	cluster string

	env     environment.Environment
	client  common.RoxctlHTTPClient
	reqBody []byte
}

// construct ensures all flag values are valid and HTTP client/request built.
func (i *imageSBOMCommand) construct(cmd *cobra.Command) error {
	var err error
	i.timeout = flags.Timeout(cmd)

	if err := imageUtils.IsValidImageString(i.image); err != nil {
		return common.ErrInvalidCommandOption.CausedBy(err)
	}

	i.client, err = i.env.HTTPClient(i.timeout)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}

	// Build HTTP request body.
	req := struct {
		Cluster   string `json:"cluster"`
		ImageName string `json:"image_name"`
		Force     bool   `json:"force"`
	}{
		Cluster:   i.cluster,
		ImageName: i.image,
		Force:     i.force,
	}

	i.reqBody, err = json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "creating request body")
	}

	return nil
}

func (i *imageSBOMCommand) GenerateSBOM() error {
	// TODO: Add retries
	// Send HTTP request and verify response status code.
	resp, err := i.client.DoReqAndVerifyStatusCode(
		imageSBOMAPIPath,
		http.MethodPost,
		http.StatusOK,
		bytes.NewReader(i.reqBody),
	)
	if err != nil {
		return errors.Wrap(err, "generating SBOM")
	}

	// Output the raw SBOM.
	_, err = io.Copy(i.env.InputOutput().Out(), resp.Body)
	if err != nil {
		return errors.Wrap(err, "writing response")
	}

	return nil
}
