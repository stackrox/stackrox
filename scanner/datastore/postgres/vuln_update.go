package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
)

const lastVulnUpdateKey = `last-vuln-update`

var (
	errNoLastUpdateTime = errors.New("no last updated time in the DB")
)

// GetLastVulnerabilityUpdate retrieves the last vulnerability update from the database.
//
// The returned time will be in the form of http.TimeFormat.
func (m *matcherMetadataStore) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	const selectTimestamp = `SELECT timestamp FROM last_vuln_update WHERE key = $1`

	var t string
	row := m.pool.QueryRow(ctx, selectTimestamp, lastVulnUpdateKey)
	err := row.Scan(&t)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, err
	}
	if err != nil || t == "" {
		return time.Time{}, errNoLastUpdateTime
	}

	timestamp, err := time.Parse(http.TimeFormat, strings.TrimSpace(t))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return timestamp, nil
}

// SetLastVulnerabilityUpdate sets the last vulnerability update timestamp.
func (m *matcherMetadataStore) SetLastVulnerabilityUpdate(ctx context.Context, timestamp time.Time) error {
	const insertTimestamp = `INSERT INTO last_vuln_update (key, timestamp) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET timestamp = $2`

	_, err := m.pool.Exec(ctx, insertTimestamp, lastVulnUpdateKey, timestamp.Format(http.TimeFormat))
	if err != nil {
		return err
	}

	return nil
}
