package main

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/indexer"
)

func sbomCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "sbom http(s)://<image-reference>",
		Short: "Generate sboms.",
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

		sbomB, found, err := scanner.GetSBOM(ctx, client.GetImageManifestID(ref))
		if !found {
			opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: false}
			ir, err := scanner.GetOrCreateImageIndex(ctx, ref, auth, opt)
			if err != nil {
				return fmt.Errorf("scanning: %w", err)
			}

			sbomB, _, err = scanner.GetSBOM(ctx, ir.GetHashId())
		}
		if err != nil {
			return fmt.Errorf("generating sbom: %w", err)
		}

		fmt.Println(string(sbomB))
		return nil
	}
	return &cmd
}
