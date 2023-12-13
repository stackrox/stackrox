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

func (m *MetadataStore) GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error) {
	const selectTimestamp = `SELECT timestamp FROM last_vuln_update WHERE key = $1`

	var t string
	row := m.pool.QueryRow(ctx, selectTimestamp, lastVulnUpdateKey)
	err := row.Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	timestamp, err := time.Parse(http.TimeFormat, strings.TrimSpace(t))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return timestamp, nil
}

func (m *MetadataStore) SetLastVulnerabilityUpdate(ctx context.Context, timestamp time.Time) error {
	const insertTimestamp = `INSERT INTO last_vuln_update(key, timestamp) VALUES($1, $2) ON CONFLICT (key) DO UPDATE SET timestamp = $2`

	_, err := m.pool.Exec(ctx, insertTimestamp, lastVulnUpdateKey, timestamp.Format(http.TimeFormat))
	if err != nil {
		return err
	}

	return nil
}
