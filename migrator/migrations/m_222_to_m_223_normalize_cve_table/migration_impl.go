package m222tom223

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
)

const (
	// Create the normalized cves table.
	createCVEsTable = `
CREATE TABLE IF NOT EXISTS cves (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cve_name        TEXT NOT NULL,
    source          TEXT NOT NULL,
    severity        TEXT NOT NULL,
    cvss_v2         FLOAT,
    cvss_v3         FLOAT,
    nvd_cvss_v3     FLOAT,
    summary         TEXT,
    link            TEXT,
    published_on    TIMESTAMPTZ,
    advisory_name   TEXT,
    advisory_link   TEXT,
    content_hash    TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (cve_name, source, content_hash)
)`

	// Create indexes on cves table.
	createCVEsNameSourceIndex = `CREATE INDEX IF NOT EXISTS cves_name_source_idx ON cves (cve_name, source)`
	createCVEsNameIndex       = `CREATE INDEX IF NOT EXISTS cves_name_idx ON cves (cve_name)`

	// Create the component_cve_edges table.
	createComponentCVEEdgesTable = `
CREATE TABLE IF NOT EXISTS component_cve_edges (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component_id            TEXT NOT NULL REFERENCES image_component_v2(id) ON DELETE CASCADE,
    cve_id                  UUID NOT NULL REFERENCES cves(id),
    is_fixable              BOOLEAN NOT NULL DEFAULT false,
    fixed_by                TEXT,
    state                   TEXT NOT NULL DEFAULT 'OBSERVED',
    first_system_occurrence TIMESTAMPTZ NOT NULL DEFAULT now(),
    fix_available_at        TIMESTAMPTZ,
    UNIQUE (component_id, cve_id)
)`

	// Create indexes on component_cve_edges table.
	createComponentCVEEdgesComponentIndex = `CREATE INDEX IF NOT EXISTS comp_cve_edges_component_idx ON component_cve_edges (component_id)`
	createComponentCVEEdgesCVEIndex       = `CREATE INDEX IF NOT EXISTS comp_cve_edges_cve_idx ON component_cve_edges (cve_id)`

	// Drop old tables.
	dropImageCVEsV2Table   = `DROP TABLE IF EXISTS image_cves_v2`
	dropImageCVEInfosTable = `DROP TABLE IF EXISTS image_cve_infos`
)

func migrate(database *types.Databases) error {
	// Create the cves table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, createCVEsTable); err != nil {
		return errors.Wrap(err, "failed to create cves table")
	}

	// Create indexes on cves table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, createCVEsNameSourceIndex); err != nil {
		return errors.Wrap(err, "failed to create cves_name_source_idx index")
	}
	if _, err := database.PostgresDB.Exec(database.DBCtx, createCVEsNameIndex); err != nil {
		return errors.Wrap(err, "failed to create cves_name_idx index")
	}

	// Create the component_cve_edges table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, createComponentCVEEdgesTable); err != nil {
		return errors.Wrap(err, "failed to create component_cve_edges table")
	}

	// Create indexes on component_cve_edges table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, createComponentCVEEdgesComponentIndex); err != nil {
		return errors.Wrap(err, "failed to create comp_cve_edges_component_idx index")
	}
	if _, err := database.PostgresDB.Exec(database.DBCtx, createComponentCVEEdgesCVEIndex); err != nil {
		return errors.Wrap(err, "failed to create comp_cve_edges_cve_idx index")
	}

	// Drop the old image_cves_v2 table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, dropImageCVEsV2Table); err != nil {
		return errors.Wrap(err, "failed to drop image_cves_v2 table")
	}

	// Drop the old image_cve_infos table.
	if _, err := database.PostgresDB.Exec(database.DBCtx, dropImageCVEInfosTable); err != nil {
		return errors.Wrap(err, "failed to drop image_cve_infos table")
	}

	return nil
}
