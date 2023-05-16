package largeobject

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"testing"

	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stretchr/testify/suite"
)

type gormUtilsTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(gormUtilsTestSuite))
}

func (s *gormUtilsTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	s.ctx = context.Background()

}

func (s *gormUtilsTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *gormUtilsTestSuite) TestMigration() {
	randomData := make([]byte, 10000)
	_, err := rand.Read(randomData)
	s.NoError(err)

	gormDB := s.db.GetGormDB()

	reader := bytes.NewBuffer(randomData)
	tx := gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los := LargeObjects{tx}
	oid, err := los.Create()
	s.NoError(err)
	err = los.Upsert(oid, reader)
	s.NoError(err)
	s.NoError(tx.Commit().Error)

	tx = gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	writer := bytes.NewBuffer([]byte{})
	s.Require().NoError(los.Get(oid, writer))

	s.Require().Equal(randomData, writer.Bytes())
	reader = bytes.NewBuffer([]byte("hi"))
	tx = gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	err = los.Upsert(oid, reader)
	s.NoError(err)
	s.NoError(tx.Commit().Error)

	tx = gormDB.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	los = LargeObjects{tx}
	writer = bytes.NewBuffer([]byte{})
	s.Require().NoError(los.Get(oid, writer))
	s.Require().Equal([]byte("hi"), writer.Bytes())
}

/*
func TestLargeObjects(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}

	skipCockroachDB(t, conn, "Server does support large objects")

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testLargeObjects(t, ctx, tx)
}

func TestLargeObjectsPreferSimpleProtocol(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgx.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}

	config.PreferSimpleProtocol = true

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}

	skipCockroachDB(t, conn, "Server does support large objects")

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testLargeObjects(t, ctx, tx)
}

func testLargeObjects(t *testing.T, ctx context.Context, tx pgx.Tx) {
	lo := tx.LargeObjects()

	id, err := lo.Create(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := lo.Open(ctx, id, pgx.LargeObjectModeRead|pgx.LargeObjectModeWrite)
	if err != nil {
		t.Fatal(err)
	}

	n, err := obj.Write([]byte("testing"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Errorf("Expected n to be 7, got %d", n)
	}

	pos, err := obj.Seek(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if pos != 1 {
		t.Errorf("Expected pos to be 1, got %d", pos)
	}

	res := make([]byte, 6)
	n, err = obj.Read(res)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != "esting" {
		t.Errorf(`Expected res to be "esting", got %q`, res)
	}
	if n != 6 {
		t.Errorf("Expected n to be 6, got %d", n)
	}

	n, err = obj.Read(res)
	if err != io.EOF {
		t.Error("Expected io.EOF, go nil")
	}
	if n != 0 {
		t.Errorf("Expected n to be 0, got %d", n)
	}

	pos, err = obj.Tell()
	if err != nil {
		t.Fatal(err)
	}
	if pos != 7 {
		t.Errorf("Expected pos to be 7, got %d", pos)
	}

	err = obj.Truncate(1)
	if err != nil {
		t.Fatal(err)
	}

	pos, err = obj.Seek(-1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if pos != 0 {
		t.Errorf("Expected pos to be 0, got %d", pos)
	}

	res = make([]byte, 2)
	n, err = obj.Read(res)
	if err != io.EOF {
		t.Errorf("Expected err to be io.EOF, got %v", err)
	}
	if n != 1 {
		t.Errorf("Expected n to be 1, got %d", n)
	}
	if res[0] != 't' {
		t.Errorf("Expected res[0] to be 't', got %v", res[0])
	}

	err = obj.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = lo.Unlink(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	_, err = lo.Open(ctx, id, pgx.LargeObjectModeRead)
	if e, ok := err.(*pgconn.PgError); !ok || e.Code != "42704" {
		t.Errorf("Expected undefined_object error (42704), got %#v", err)
	}
}

func TestLargeObjectsMultipleTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, os.Getenv("PGX_TEST_DATABASE"))
	if err != nil {
		t.Fatal(err)
	}

	skipCockroachDB(t, conn, "Server does support large objects")

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	lo := tx.LargeObjects()

	id, err := lo.Create(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	obj, err := lo.Open(ctx, id, pgx.LargeObjectModeWrite)
	if err != nil {
		t.Fatal(err)
	}

	n, err := obj.Write([]byte("testing"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Errorf("Expected n to be 7, got %d", n)
	}

	// Commit the first transaction
	err = tx.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// IMPORTANT: Use the same connection for another query
	query := `select n from generate_series(1,10) n`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()

	// Start a new transaction
	tx2, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	lo2 := tx2.LargeObjects()

	// Reopen the large object in the new transaction
	obj2, err := lo2.Open(ctx, id, pgx.LargeObjectModeRead|pgx.LargeObjectModeWrite)
	if err != nil {
		t.Fatal(err)
	}

	pos, err := obj2.Seek(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if pos != 1 {
		t.Errorf("Expected pos to be 1, got %d", pos)
	}

	res := make([]byte, 6)
	n, err = obj2.Read(res)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != "esting" {
		t.Errorf(`Expected res to be "esting", got %q`, res)
	}
	if n != 6 {
		t.Errorf("Expected n to be 6, got %d", n)
	}

	n, err = obj2.Read(res)
	if err != io.EOF {
		t.Error("Expected io.EOF, go nil")
	}
	if n != 0 {
		t.Errorf("Expected n to be 0, got %d", n)
	}

	pos, err = obj2.Tell()
	if err != nil {
		t.Fatal(err)
	}
	if pos != 7 {
		t.Errorf("Expected pos to be 7, got %d", pos)
	}

	err = obj2.Truncate(1)
	if err != nil {
		t.Fatal(err)
	}

	pos, err = obj2.Seek(-1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if pos != 0 {
		t.Errorf("Expected pos to be 0, got %d", pos)
	}

	res = make([]byte, 2)
	n, err = obj2.Read(res)
	if err != io.EOF {
		t.Errorf("Expected err to be io.EOF, got %v", err)
	}
	if n != 1 {
		t.Errorf("Expected n to be 1, got %d", n)
	}
	if res[0] != 't' {
		t.Errorf("Expected res[0] to be 't', got %v", res[0])
	}

	err = obj2.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = lo2.Unlink(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	_, err = lo2.Open(ctx, id, pgx.LargeObjectModeRead)
	if e, ok := err.(*pgconn.PgError); !ok || e.Code != "42704" {
		t.Errorf("Expected undefined_object error (42704), got %#v", err)
	}
}
*/
