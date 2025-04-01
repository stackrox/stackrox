package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/indexer"
)

func sbomCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "sbom http(s)://<image-reference>",
		Short: "Generate SBOM for an image. Will index the image if not already indexed.",
		Args:  cobra.ExactArgs(1),
	}

	flags := cmd.PersistentFlags()
	basicAuth := flags.String(
		"auth",
		"",
		fmt.Sprintf("Use the specified basic auth credentials (warning: debug "+
			"only and unsafe, use env var %s).", authn.BasicAuthSetting))
	imageDigest := flags.String(
		"digest",
		"",
		"Use the specified image digest in "+
			"the image manifest ID. The default is to retrieve the image digest from "+
			"the registry and use that.")
	skipIndexing := flags.Bool(
		"skip-indexing",
		false,
		"Do not index the image when an existing index report is not found",
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}
		// Extract basic auth username and password.
		auth, err := authn.ParseBasic(*basicAuth)
		if err != nil {
			return err
		}
		// Get the image digest, from the URL or command option.
		imageURL := args[0]
		ref, err := indexer.GetDigestFromURL(imageURL, auth)
		if err != nil {
			return fmt.Errorf("failed to retrieve image hash id: %w", err)
		}
		if *imageDigest == "" {
			*imageDigest = ref.DigestStr()
			log.Printf("image digest: %s", *imageDigest)
		}
		if *imageDigest != ref.DigestStr() {
			log.Printf("WARNING: the actual image digest %q is different from %q",
				ref.DigestStr(), *imageDigest)
		}

		uri := imageURL + "-" + uuid.NewV4().String()
		// Attempt to generate the SBOM from an existing index report.
		sbom, found, err := scanner.GetSBOM(ctx, imageURL, ref, uri)
		if err != nil {
			return fmt.Errorf("generating sbom: %w", err)
		}
		if found {
			fmt.Println(string(sbom))
			return nil
		}
		if *skipIndexing {
			return fmt.Errorf("index report not found for %q", ref)
		}

		// Index the image.
		var ir *v4.IndexReport
		// TODO(ROX-23898): add flag for skipping TLS verification.
		opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: false}
		ir, err = scanner.GetOrCreateImageIndex(ctx, ref, auth, opt)
		if err != nil {
			return fmt.Errorf("indexing: %w", err)
		}
		if ir == nil || !ir.GetSuccess() {
			return errors.New("index report missing or unsuccessful")
		}

		// Re-attempt to generate the SBOM.
		sbom, found, err = scanner.GetSBOM(ctx, imageURL, ref, uri)
		if err != nil {
			return fmt.Errorf("generating sbom: %w", err)
		}
		if !found {
			return fmt.Errorf("index report still not found for %q", ref)
		}

		fmt.Println(string(sbom))
		return nil
	}

	return &cmd
}
