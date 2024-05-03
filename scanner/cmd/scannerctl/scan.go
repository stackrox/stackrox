package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/indexer"
)

// scanCmd creates the scan command.
func scanCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "scan http(s)://<image-reference>",
		Short: "Perform vulnerability scans.",
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
	indexOnly := flags.Bool(
		"index-only",
		false,
		"Only index the specified image")
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
		var report any
		opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: false}
		if *indexOnly {
			var ir *v4.IndexReport
			// TODO(ROX-23898): add flag for skipping TLS verification.
			ir, err = scanner.GetOrCreateImageIndex(ctx, ref, auth, opt)
			report = ir
		} else {
			var vr *v4.VulnerabilityReport
			// TODO(ROX-23898): add flag for skipping TLS verification.
			vr, err = scanner.IndexAndScanImage(ctx, ref, auth, opt)
			report = vr
		}

		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return &cmd
}
