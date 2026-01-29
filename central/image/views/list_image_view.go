package views

import "time"

// ListImageView represents the minimal fields needed for ListImage queries.
// This view maps directly to PostgreSQL columns, avoiding deserialization of
// the serialized bytea column which contains the full Image proto (~100KB).
// Only the fields needed for ListImage are fetched (~150 bytes).
// Note: Priority is not fetched as it's a derived field and will be set by the ranker.
type ListImageView struct {
	ID              string     `db:"id"`
	NameRegistry    string     `db:"name_registry"`
	NameRemote      string     `db:"name_remote"`
	NameTag         string     `db:"name_tag"`
	ComponentCount  *int32     `db:"components"`
	CveCount        *int32     `db:"cves"`
	FixableCveCount *int32     `db:"fixablecves"`
	Created         *time.Time `db:"metadata_v1_created"`
	LastUpdated     *time.Time `db:"lastupdated"`
}
