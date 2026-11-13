package main

// One-shot content blob helpers: -store-all / -fetch-any.
//
// One-shot blobs (vendor trees, ui deps, built UI, go-binaries tars)
// differ fundamentally from cache entries: they are consumed by jobs in
// ARBITRARY regions, often within the same run, so replication lag
// between regional stores makes a single-account write miss for
// consumers elsewhere. Storing to all replicas (~3x write cost, trivial
// at blob pricing) and fetching with cross-replica fallback makes them
// region-agnostic.

import (
	"fmt"
	"net/http"
	"time"
	"os"
)

// replicaAccounts lists the storage accounts holding one-shot content.
// Azure-only by construction; dir test mode is single-store.
var replicaAccounts = []string{"cigocache", "cigocacheeast", "cigochachecentral"}

// storeAll writes the file's bytes to every replica. Failures on
// individual replicas are logged but not fatal — a majority of replicas
// holding the content is strictly better than aborting.
func storeAll(key, file string) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	stored, failed := 0, 0
	for _, acct := range replicaAccounts {
		b := azureBackendFor(acct)
		if err := b.Put(key, body); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "store-all: %s failed: %v\n", acct, err)
			continue
		}
		stored++
	}
	fmt.Printf("store-all: %s -> %d/%d replicas (%d bytes)\n", key, stored, len(replicaAccounts), len(body))
	if stored == 0 {
		return fmt.Errorf("all %d replicas failed", len(replicaAccounts))
	}
	return nil
}

// fetchAny fetches the key, trying the region-selected account first
// (fast path: same-region read), then every replica. Returns nil
// (exit 1) only if no replica has it.
func fetchAny(key, file string) error {
	// Fast path: the selected/primary account (region-probed).
	if b, err := newBackend(); err == nil {
		if body, gerr := b.Get(key); gerr == nil && body != nil {
			if werr := os.WriteFile(file, body, 0o644); werr != nil {
				return werr
			}
			fmt.Printf("fetch-any: %s from selected account (%d bytes)\n", key, len(body))
			return nil
		}
	}
	for _, acct := range replicaAccounts {
		b := azureBackendFor(acct)
		body, err := b.Get(key)
		if err != nil || body == nil {
			continue
		}
		if werr := os.WriteFile(file, body, 0o644); werr != nil {
			return werr
		}
		fmt.Printf("fetch-any: %s from %s (%d bytes)\n", key, acct, len(body))
		return nil
	}
	return fmt.Errorf("no replica has %s", key)
}

// azureBackendFor returns a backend pinned to a specific account
// (bypassing region selection). Returns nil in dir test mode.
func azureBackendFor(account string) Backend {
	if os.Getenv("GOCACHE_SERVE_BACKEND") == "dir" {
		if b, err := newBackend(); err == nil {
			return b
		}
		return nil
	}
	return &AzureBackend{
		account:      account,
		container:    containerFromEnv(),
		tenantID:     os.Getenv("AZURE_TENANT_ID"),
		clientID:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		client: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 16,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}
