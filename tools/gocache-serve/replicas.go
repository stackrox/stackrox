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
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"
	"os"
	"strings"
	"sync"
)

// replicaAccounts lists the storage accounts holding one-shot content.
// Azure-only by construction; dir test mode is single-store.
var replicaAccounts = []string{"cigocache", "cigocacheeast", "cigochachecentral"}

// storeAll ensures the file's content is present and hash-marked on
// every replica — IDEMPOTENTLY and IN PARALLEL:
//
//   - Each replica's existing marker is checked first (a tiny GET).
//     If it already matches this content's hash, that replica is
//     skipped: warm runs (unchanged binaries, identical vendor trees)
//     upload nothing instead of re-pushing gigabytes (measured: the
//     binaries store step was 72s of redundant upload on every run).
//   - Replicas needing upload are written concurrently (independent
//     accounts; cold-case wall time = one upload, not three).
//   - The marker is written AFTER the content per replica, so its
//     presence implies a deliberate, complete store.
func storeAll(key, file string) error {
	body, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(maybeGunzip(body))
	want := fmt.Sprintf("%x", sum)

	type result struct {
		acct   string
		skipped bool
		err    error
	}
	results := make(chan result, len(replicaAccounts))
	var wg sync.WaitGroup
	for _, acct := range replicaAccounts {
		wg.Add(1)
		go func(acct string) {
			defer wg.Done()
			b := azureBackendFor(acct)
			if b == nil {
				results <- result{acct, false, nil}
				return
			}
			// Marker-only check: if the stored marker equals our hash,
			// this replica already has exactly this content.
			if marker, err := b.Get(key + ".sha256"); err == nil && marker != nil {
				if strings.TrimSpace(string(marker)) == want {
					results <- result{acct, true, nil}
					return
				}
			}
			if err := b.Put(key, body); err != nil {
				results <- result{acct, false, err}
				return
			}
			if err := b.Put(key+".sha256", []byte(want+"\n")); err != nil {
				results <- result{acct, false, err}
				return
			}
			results <- result{acct, false, nil}
		}(acct)
	}
	wg.Wait()
	close(results)

	stored, skipped, failed := 0, 0, 0
	for r := range results {
		if r.err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "store-all: %s failed: %v\n", r.acct, r.err)
		} else if r.skipped {
			skipped++
		} else {
			stored++
		}
	}
	fmt.Printf("store-all: %s -> %d stored, %d already-current, %d failed of %d replicas (%d bytes, sha256 %s)\n",
		key, stored, skipped, failed, len(replicaAccounts), len(body), want[:16])
	if stored+skipped == 0 {
		return fmt.Errorf("all %d replicas failed", len(replicaAccounts))
	}
	return nil
}

// fetchAny fetches the key, trying the region-selected account first
// (fast path: same-region read), then every replica. Content is only
// served if its hash marker (KEY.sha256, written by storeAll) exists
// and matches — a blob without a verifiable marker is UNTRUSTED:
// either stored before marker support (like the poisoned Nov-12 vendor
// tree) or interrupted mid-write. Untrusted content is a miss; the
// caller regenerates and the post-build store replaces it.
func fetchAny(key, file string) error {
	// Fast path: the selected/primary account (region-probed).
	if b, err := newBackend(); err == nil && b != nil {
		if ok, body := fetchVerified(b, key); ok {
			if werr := os.WriteFile(file, body, 0o644); werr != nil {
				return werr
			}
			fmt.Printf("fetch-any: %s from selected account (%d bytes, verified)\n", key, len(body))
			return nil
		}
	}
	for _, acct := range replicaAccounts {
		b := azureBackendFor(acct)
		if b == nil {
			continue
		}
		if ok, body := fetchVerified(b, key); ok {
			if werr := os.WriteFile(file, body, 0o644); werr != nil {
				return werr
			}
			fmt.Printf("fetch-any: %s from %s (%d bytes, verified)\n", key, acct, len(body))
			return nil
		}
	}
	return fmt.Errorf("no replica has a verified %s", key)
}

// fetchVerified returns the content only if its hash marker matches.
// Marker missing → untrusted (miss). Marker present but mismatched →
// corrupt (miss, loudly).
func fetchVerified(b Backend, key string) (bool, []byte) {
	body, err := b.Get(key)
	if err != nil || body == nil {
		return false, nil
	}
	marker, err := b.Get(key + ".sha256")
	if err != nil || marker == nil {
		fmt.Fprintf(os.Stderr, "fetch-any: %s has no hash marker — untrusted, treating as miss\n", key)
		return false, nil
	}
	sum := sha256.Sum256(body)
	want := strings.TrimSpace(string(marker))
	if got := fmt.Sprintf("%x", sum); got != want {
		fmt.Fprintf(os.Stderr, "fetch-any: %s hash mismatch (got %s, marker %s) — corrupt, treating as miss\n", key, got[:16], want[:16])
		return false, nil
	}
	return true, body
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
