package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/indexer"
)

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
		Use:   "scale <registry> [OPTIONS]",
		Short: "Perform scale tests via querying for the first N images in the given repository.",
		Args:  cobra.ExactArgs(1),
	}
	flags := cmd.PersistentFlags()
	basicAuth := flags.String(
		"auth",
		"",
		fmt.Sprintf("Use the specified basic auth credentials (warning: debug "+
			"only and unsafe, use env var %s).", authn.BasicAuthSetting))
	images := flags.Int(
		"images",
		1000,
		"Specify the number of images from the given repository to scan")
	workers := flags.Int(
		"workers",
		15,
		"Specify the number of parallel scans")
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
		repoName := args[0]
		repo, err := name.NewRepository(repoName, name.StrictValidation)
		if err != nil {
			panic("programmer error")
		}
		puller, err := remote.NewPuller(remote.WithAuth(auth))
		if err != nil {
			log.Fatalf("creating puller: %v", err)
		}
		lister, err := puller.Lister(ctx, repo)
		if err != nil {
			log.Fatalf("creating lister: %v", err)
		}
		tags := make([]string, 0, *images)
		for lister.HasNext() {
			ts, err := lister.Next(ctx)
			if err != nil {
				log.Fatalf("listing tags: %v", err)
			}
			tags = append(tags, ts.Tags...)
			if len(tags) >= *images {
				tags = tags[:*images]
				break
			}
		}
		log.Printf("scale testing with %d images", len(tags))

		tagsC := make(chan string)
		go func() {
			for _, tag := range tags {
				tagsC <- tag
			}
			close(tagsC)
		}()

		var stats scaleStats
		var wg sync.WaitGroup
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for tag := range tagsC {
					ref, err := name.ParseReference(repoName+":"+tag, name.StrictValidation)
					if err != nil {
						stats.preFailure.Add(1)
						log.Printf("could not parse reference for tag %s: %v", tag, err)
						return
					}
					d, err := indexer.GetDigestFromReference(ref, auth)
					if err != nil {
						stats.preFailure.Add(1)
						log.Printf("could not get digest for image %v: %v", ref, err)
						return
					}
					err = doWithTimeout(ctx, 5*time.Minute, func(ctx context.Context) error {
						log.Printf("indexing image %v", ref)
						_, err := scanner.GetOrCreateImageIndex(ctx, d, auth)
						if err != nil {
							stats.indexFailure.Add(1)
							return fmt.Errorf("indexing image %v: %w", ref, err)
						}
						stats.indexSuccess.Add(1)

						log.Printf("matching image %v", ref)
						// Though this method both indexes and matches, we know the indexing has already completed,
						// and this method will just verify the index still exists. We don't account for
						// this verification's potential failures at this time.
						_, err = scanner.IndexAndScanImage(ctx, d, auth)
						if err != nil {
							stats.matchFailure.Add(1)
							return fmt.Errorf("matching image %v: %w", ref, err)
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

func doWithTimeout(ctx context.Context, timeout time.Duration, f func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f(ctx)
}
