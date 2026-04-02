package m222tom223

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
)

const (
	// Create the normalized cves table.
	createCVEsTable = `
CREATE TABLE IF NOT EXISTS cves (
    id          VARCHAR PRIMARY KEY,
    cvename     VARCHAR NOT NULL,
    source      VARCHAR NOT NULL,
    severity    VARCHAR NOT NULL,
    cvssv2      NUMERIC,
    cvssv3      NUMERIC,
    nvdcvssv3   NUMERIC,
    publishedon TIMESTAMPTZ,
    createdat   TIMESTAMPTZ,
    serialized  BYTEA,
    UNIQUE (cvename, source)
)`

	// Create indexes on cves table.
	createCVEsNameSourceIndex = `CREATE INDEX IF NOT EXISTS cves_name_source_idx ON cves (cvename, source)`
	createCVEsNameIndex       = `CREATE INDEX IF NOT EXISTS cves_name_idx ON cves (cvename)`

	// Create the component_cve_edges table.
	createComponentCVEEdgesTable = `
CREATE TABLE IF NOT EXISTS component_cve_edges (
    id          VARCHAR PRIMARY KEY,
    componentid VARCHAR NOT NULL REFERENCES image_component_v2(id) ON DELETE CASCADE,
    cveid       VARCHAR NOT NULL REFERENCES cves(id),
    isfixable   BOOLEAN NOT NULL DEFAULT false,
    fixedby     VARCHAR,
    state       VARCHAR NOT NULL DEFAULT 'OBSERVED',
    serialized  BYTEA,
    UNIQUE (componentid, cveid)
)`

	// Create indexes on component_cve_edges table.
	createComponentCVEEdgesComponentIndex = `CREATE INDEX IF NOT EXISTS comp_cve_edges_component_idx ON component_cve_edges (componentid)`
	createComponentCVEEdgesCVEIndex       = `CREATE INDEX IF NOT EXISTS comp_cve_edges_cve_idx ON component_cve_edges (cveid)`

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
