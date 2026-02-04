package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/jackc/pgx/v5/pgxpool"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/pkg/ctxlock/v2"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stackrox/rox/scanner/matcher"
)

const dbPasswordEnvVar = "ROX_SCANNERCTL_DB_PASSWORD"

// ccVulnReportCmd creates the ccvulnreport command.
func ccVulnReportCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "ccvulnreport http(s)://<image-reference>",
		Short: "Output the raw ClairCore vulnerability report for an already-indexed image.",
		Long: "Output the raw ClairCore vulnerability report for an already-indexed image.\n" +
			"The image reference must include a digest (e.g. https://registry/image:tag@sha256:...).\n" +
			"Use 'scannerctl scan' or 'roxctl image scan' to index an image and retrieve its digest.",
		Args: cobra.ExactArgs(1),
	}

	flags := cmd.PersistentFlags()
	dbHost := flags.String(
		"db-host",
		"127.0.0.1",
		"Database host (assumes an active port-forward to the scanner-v4-db pod).")
	dbPort := flags.String(
		"db-port",
		"5432",
		"Database port.")
	dbPassword := flags.String(
		"db-password",
		"",
		fmt.Sprintf("Database password (warning: debug only and unsafe, use env var %s). "+
			"If unset, the password is retrieved from the scanner-v4-db-password k8s secret via kubectl.", dbPasswordEnvVar))

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Parse the image reference — must include a digest.
		imageURL := args[0]
		parsedURL, err := url.Parse(imageURL)
		if err != nil {
			return fmt.Errorf("invalid image URL: %w", err)
		}
		var nameOpts []name.Option
		if parsedURL.Scheme == "http" {
			nameOpts = append(nameOpts, name.Insecure)
		}
		imageRef := strings.TrimPrefix(imageURL, parsedURL.Scheme+"://")
		ref, err := name.NewDigest(imageRef, nameOpts...)
		if err != nil {
			return fmt.Errorf("image reference must include a digest (e.g. registry/image:tag@sha256:...): %w", err)
		}

		// Get DB password from flag, env var, or k8s secret (in that order).
		password, err := getScannerDBPassword(ctx, *dbPassword)
		if err != nil {
			return err
		}

		poolCfg, err := pgxpool.ParseConfig(fmt.Sprintf(
			"host=%s port=%s user=postgres sslmode=disable pool_max_conns=5 client_encoding=UTF8",
			*dbHost, *dbPort))
		if err != nil {
			return fmt.Errorf("failed to parse database config: %w", err)
		}
		poolCfg.ConnConfig.Password = password

		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer pool.Close()

		log.Printf("Fetching index report from database...")
		indexerStore, err := ccpostgres.InitPostgresIndexerStore(ctx, pool, false)
		if err != nil {
			return fmt.Errorf("failed to initialize indexer store: %w", err)
		}
		defer indexerStore.Close(ctx)

		hashID := fmt.Sprintf("/v4/containerimage/%s", ref.DigestStr())
		manifestDigest, err := indexer.CreateManifestDigest(hashID)
		if err != nil {
			return fmt.Errorf("failed to create manifest digest: %w", err)
		}
		ccIndexReport, found, err := indexerStore.IndexReport(ctx, manifestDigest)
		if err != nil {
			return fmt.Errorf("failed to get index report: %w", err)
		}
		if !found {
			externalStore, err := postgres.InitPostgresExternalIndexStore(ctx, pool, false)
			if err != nil {
				return fmt.Errorf("failed to initialize external index store: %w", err)
			}
			ccIndexReport, found, err = externalStore.GetIndexReport(ctx, hashID)
			if err != nil {
				return fmt.Errorf("failed to get external index report: %w", err)
			}
			if !found {
				return fmt.Errorf("index report not found for image %s — ensure the image is indexed before running this command", ref.DigestStr())
			}
			log.Printf("Found external index report with %d packages", len(ccIndexReport.Packages))
		} else {
			log.Printf("Found index report with %d packages", len(ccIndexReport.Packages))
		}

		store, err := postgres.InitPostgresMatcherStore(ctx, pool, false)
		if err != nil {
			return fmt.Errorf("failed to initialize matcher store: %w", err)
		}

		locker, err := ctxlock.New(ctx, pool)
		if err != nil {
			return fmt.Errorf("failed to create locker: %w", err)
		}
		defer locker.Close(ctx)

		libVuln, err := libvuln.New(ctx, &libvuln.Options{
			Store:                    store,
			Locker:                   locker,
			MatcherNames:             matcher.GetMatcherNames(),
			Enrichers:                matcher.GetEnrichers(),
			UpdateRetention:          libvuln.DefaultUpdateRetention,
			DisableBackgroundUpdates: true,
			Client:                   &http.Client{Transport: httputil.DenyTransport},
		})
		if err != nil {
			return fmt.Errorf("failed to create libvuln: %w", err)
		}
		defer libVuln.Close(ctx)

		log.Printf("Scanning for vulnerabilities...")
		vulnReport, err := libVuln.Scan(ctx, ccIndexReport)
		if err != nil {
			return fmt.Errorf("failed to scan: %w", err)
		}
		log.Printf("Found %d vulnerabilities", len(vulnReport.Vulnerabilities))

		reportJSON, err := json.MarshalIndent(vulnReport, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return &cmd
}

// getScannerDBPassword returns the scanner DB password from the flag, env var,
// or k8s secret (in that order).
func getScannerDBPassword(ctx context.Context, flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if p := os.Getenv(dbPasswordEnvVar); p != "" {
		return p, nil
	}
	log.Printf("db-password unspecified: retrieving from k8s secret (use %s to set directly)", dbPasswordEnvVar)
	out, err := exec.CommandContext(ctx, "kubectl", "get", "secret", "scanner-v4-db-password",
		"-o", "jsonpath={.data.password}").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get db password secret: %w", err)
	}
	p, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
	if err != nil {
		return "", fmt.Errorf("failed to decode password: %w", err)
	}
	return strings.TrimSpace(string(p)), nil
}
