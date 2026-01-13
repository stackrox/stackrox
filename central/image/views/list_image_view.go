package views

import "time"

// ListImageView represents the minimal fields needed for ListImage queries.
// This view maps directly to PostgreSQL columns, avoiding deserialization of
// the serialized bytea column which contains the full Image proto (~100KB).
// Only the 8 fields needed for ListImage are fetched (~150 bytes).
type ListImageView struct {
	ID              string     `db:"id"`
	Name            string     `db:"name_fullname"`
	ComponentCount  *int32     `db:"scanstats_componentcount"`
	CveCount        *int32     `db:"scanstats_cvecount"`
	FixableCveCount *int32     `db:"scanstats_fixablecvecount"`
	Created         *time.Time `db:"metadata_v1_created"`
	LastUpdated     *time.Time `db:"lastupdated"`
	Priority        int64      `db:"priority"`
}
