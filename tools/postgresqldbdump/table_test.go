//go:build sql_integration

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-jose/go-jose/v3/json"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	postgresTest "github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/multitest/postgres"
	"github.com/stretchr/testify/suite"
)

type testSuite struct {
	suite.Suite

	ctx    context.Context
	testDB *pgtest.TestPostgres
	store  postgresTest.Store
}

func TestToolTable(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
	s.store = postgresTest.New(s.testDB.DB)
}

func (s *testSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *testSuite) TestDumpScan() {
	var testStructs []*storage.TestStruct
	testStructIDs := set.NewStringSet()
	for i := 0; i < 4; i++ {
		testStruct := &storage.TestStruct{}
		s.NoError(testutils.FullInit(testStruct, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		testStructs = append(testStructs, testStruct)
		testStructIDs.Add(testStruct.GetKey1())
	}

	s.NoError(s.store.UpsertMany(s.ctx, testStructs))

	// Get DB Dump
	buf := bytes.NewBuffer(nil)
	cmd := exec.Command("pg_dump", "-d", s.testDB.DB.Config().ConnConfig.Database, "-U", s.testDB.Config().ConnConfig.User, "-Fc")
	cmd.Stdout = buf
	s.NoError(pgadmin.ExecutePostgresCmd(cmd))

	// Run the tool
	tmpDir, err := os.MkdirTemp("", "postgresqldbdump")
	s.NoError(err)
	s.NoError(pgRestore(buf, tmpDir))

	// Verify
	file, err := os.Open(filepath.Join(tmpDir, "test_structs.json"))
	s.NoError(err)
	var targets []map[string]map[string]any
	dc := json.NewDecoder(file)
	s.NoError(dc.Decode(&targets))
	s.Len(targets, 4)
	for _, t := range targets {
		s.Contains(t, "fields")
		fields := t["fields"]
		s.Contains(t, "serialized")
		serialized := t["serialized"]
		s.Contains(fields, "key1")
		s.Contains(testStructIDs, fields["key1"])
		s.Contains(serialized, "key1")
		s.Contains(testStructIDs, serialized["key1"])
	}
	s.NoError(file.Close())
}
