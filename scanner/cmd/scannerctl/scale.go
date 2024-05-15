package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	pkgauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/fixtures"
	"github.com/stackrox/rox/scanner/indexer"
)

var scanTimeout = env.ScanTimeout.DurationSetting()

// scaleStats specifies the stats we want to track when performing scale tests.
type scaleStats struct {
	preFailure atomic.Int64

	indexSuccess atomic.Int64
	indexFailure atomic.Int64

	matchSuccess atomic.Int64
	matchFailure atomic.Int64
}

func (s *scaleStats) String() string {
	var ret strings.Builder
	ret.WriteRune('\n')
	ret.WriteString(fmt.Sprintf("pre-scanning failure: %d\n", s.preFailure.Load()))
	ret.WriteString(fmt.Sprintf("index success: %d\n", s.indexSuccess.Load()))
	ret.WriteString(fmt.Sprintf("index failure: %d\n", s.indexFailure.Load()))
	ret.WriteString(fmt.Sprintf("match success: %d\n", s.matchSuccess.Load()))
	ret.WriteString(fmt.Sprintf("match failure: %d\n", s.matchFailure.Load()))
	ret.WriteRune('\n')
	return ret.String()
}

// scaleCmd creates the scale command.
func scaleCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "scale [OPTIONS]",
		Short: "Perform scale tests via querying for the first N images in the given repository.",
	}
	flags := cmd.PersistentFlags()
	basicAuth := flags.String(
		"auth",
		"",
		fmt.Sprintf("Use the specified basic auth credentials (warning: debug "+
			"only and unsafe, use env var %s).", authn.BasicAuthSetting))
	repository := flags.String(
		"repository",
		"",
		"Specify the repository from which to pull images (ex: quay.io/stackrox-io/scanner-v4)")
	images := flags.Int(
		"images",
		1000,
		"Specify the number of images from the given repository to scan (only used when repository is set)")
	workers := flags.Int(
		"workers",
		15,
		"Specify the number of parallel scans")
	indexOnly := flags.Bool(
		"index-only",
		false,
		"Only index the specified image")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Extract basic auth username and password.
		auth, err := authn.ParseBasic(*basicAuth)
		if err != nil {
			return err
		}

		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		var refs []name.Reference
		if *repository != "" {
			refs, err = references(ctx, auth, *repository, *images)
		} else {
			refs, err = fixtures.References()
		}
		if err != nil {
			return fmt.Errorf("fetching image references: %w", err)
		}

		log.Printf("scale testing with %d images with timeout %v (can be changed with %s)", len(refs), scanTimeout, env.ScanTimeout.EnvVar())

		refsC := make(chan name.Reference)
		go func() {
			for _, ref := range refs {
				refsC <- ref
			}
			close(refsC)
		}()

		var stats scaleStats
		var wg sync.WaitGroup
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ref := range refsC {
					d, err := indexer.GetDigestFromReference(ref, auth)
					if err != nil {
						stats.preFailure.Add(1)
						log.Printf("could not get digest for image %v: %v", ref, err)
						continue
					}
					err = doWithTimeout(ctx, scanTimeout, func(ctx context.Context) error {
						log.Printf("indexing image %v", ref)
						// TODO(ROX-23898): add flag for skipping TLS verification.
						opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: false}
						_, err := scanner.GetOrCreateImageIndex(ctx, d, auth, opt)
						if err != nil {
							stats.indexFailure.Add(1)
							return fmt.Errorf("indexing: %w", err)
						}
						stats.indexSuccess.Add(1)

						if *indexOnly {
							return nil
						}

						log.Printf("matching image %v", ref)
						// Though this method both indexes and matches, we know the indexing has already completed,
						// and this method will just verify the index still exists. We don't account for
						// this verification's potential failures at this time.
						// TODO(ROX-23898): add flag for skipping TLS verification.
						_, err = scanner.IndexAndScanImage(ctx, d, auth, opt)
						if err != nil {
							stats.matchFailure.Add(1)
							return fmt.Errorf("matching: %w", err)
						}
						stats.matchSuccess.Add(1)

						return nil
					})
					if err != nil {
						log.Printf("error scanning image %v: %v", ref, err)
					}
				}
			}()
		}

		wg.Wait()

		log.Printf("scale tests complete: %v", &stats)

		return nil
	}
	return &cmd
}

func references(ctx context.Context, auth pkgauthn.Authenticator, repository string, n int) ([]name.Reference, error) {
	repo, err := name.NewRepository(repository, name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("validating repository: %w", err)
	}
	puller, err := remote.NewPuller(remote.WithAuth(auth))
	if err != nil {
		return nil, fmt.Errorf("creating puller: %w", err)
	}
	lister, err := puller.Lister(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("creating lister: %w", err)
	}

	refs := make([]name.Reference, 0, n)
ListTags:
	for lister.HasNext() {
		ts, err := lister.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %w", err)
		}

		for _, tag := range ts.Tags {
			ref, err := name.ParseReference(repository+":"+tag, name.StrictValidation)
			if err != nil {
				return nil, err
			}

			refs = append(refs, ref)

			if len(refs) == cap(refs) {
				break ListTags
			}
		}
	}

	return refs, nil
}

func doWithTimeout(ctx context.Context, timeout time.Duration, f func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f(ctx)
}
