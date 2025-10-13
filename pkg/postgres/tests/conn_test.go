package tests

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	pkgmocks "github.com/stackrox/rox/pkg/mocks/github.com/jackc/pgx/v5/mocks"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake = errors.New("fake error")
)

func TestAlertDataStore(t *testing.T) {
	suite.Run(t, new(postgresConnTestSuite))
}

type postgresConnTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockPgxPoolConn *mocks.MockPgxPoolConn

	mockTx *pkgmocks.MockTx

	conn *postgres.Conn

	tx *postgres.Tx
}

func (s *postgresConnTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *postgresConnTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *postgresConnTestSuite) SetupTest() {
	s.mockPgxPoolConn = mocks.NewMockPgxPoolConn(s.mockCtrl)
	s.conn = &postgres.Conn{PgxPoolConn: s.mockPgxPoolConn}
	s.mockTx = pkgmocks.NewMockTx(s.mockCtrl)
	s.tx = &postgres.Tx{}
	s.tx.Tx = s.mockTx
}

func (s *postgresConnTestSuite) TestConnBegin() {
	// Without tx
	s.mockPgxPoolConn.EXPECT().Begin(gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context) (pgx.Tx, error) {
		return s.mockTx, nil
	})
	tx1, err := s.conn.Begin(context.Background())
	s.NoError(err)
	s.NotNil(tx1)

	// With tx
	ctxWithTx := postgres.ContextWithTx(context.Background(), tx1)
	tx2, err := s.conn.Begin(ctxWithTx)
	s.NoError(err)
	s.Same(tx1.Tx, tx2.Tx)

	// Error
	s.mockPgxPoolConn.EXPECT().Begin(gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context) (pgx.Tx, error) {
		return nil, errFake
	})
	tx3, err := s.conn.Begin(context.Background())
	s.Equal(errFake, err)
	s.Nil(tx3)
}

func (s *postgresConnTestSuite) TestConnExec() {
	expectedCt := pgconn.NewCommandTag("signature")
	errFunc := func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, errFake
	}
	successFunc := func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
		return expectedCt, nil
	}

	ctxWithTx := postgres.ContextWithTx(context.Background(), s.tx)

	// With Tx
	s.mockTx.EXPECT().Exec(gomock.Any(), "command1").Times(1).DoAndReturn(successFunc)
	ct, err := s.conn.Exec(ctxWithTx, "command1")
	s.NoError(err)
	s.Equal(expectedCt, ct)

	// Without Tx
	s.mockPgxPoolConn.EXPECT().Exec(gomock.Any(), "command2").Times(1).DoAndReturn(successFunc)
	ct, err = s.conn.Exec(context.Background(), "command2")
	s.NoError(err)
	s.Equal(expectedCt, ct)

	// Error Handling with Tx
	s.mockTx.EXPECT().Exec(gomock.Any(), "command3").Times(1).DoAndReturn(errFunc)
	ct, err = s.conn.Exec(ctxWithTx, "command3")
	s.Equal(errFake, err)
	s.Empty(ct)

	// Error Handling without Tx
	s.mockPgxPoolConn.EXPECT().Exec(gomock.Any(), "command4").Times(1).DoAndReturn(errFunc)
	ct, err = s.conn.Exec(context.Background(), "command4")
	s.Equal(errFake, err)
	s.Empty(ct)
}

func (s *postgresConnTestSuite) TestConnQuery() {
	expectedRows := pkgmocks.NewRows(nil).ToPgxRows()
	errFunc := func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
		return nil, errFake
	}
	successFunc := func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
		return expectedRows, nil
	}

	ctxWithTx := postgres.ContextWithTx(context.Background(), s.tx)

	// With Tx
	s.mockTx.EXPECT().Query(gomock.Any(), "command1").Times(1).DoAndReturn(successFunc)
	rows, err := s.conn.Query(ctxWithTx, "command1")
	s.NoError(err)
	s.Equal(expectedRows, rows.Rows)

	// Without Tx
	s.mockPgxPoolConn.EXPECT().Query(gomock.Any(), "command2").Times(1).DoAndReturn(successFunc)
	rows, err = s.conn.Query(context.Background(), "command2")
	s.NoError(err)
	s.Equal(expectedRows, rows.Rows)

	// Error Handling with Tx
	s.mockTx.EXPECT().Query(gomock.Any(), "command3").Times(1).DoAndReturn(errFunc)
	rows, err = s.conn.Query(ctxWithTx, "command3")
	s.Equal(errFake, err)
	s.Nil(rows)

	// Error Handling without Tx
	s.mockPgxPoolConn.EXPECT().Query(gomock.Any(), "command4").Times(1).DoAndReturn(errFunc)
	rows, err = s.conn.Query(context.Background(), "command4")
	s.Equal(errFake, err)
	s.Nil(rows)
}

func (s *postgresConnTestSuite) TestConnQueryRow() {
	expectedRow := pkgmocks.NewRows(nil).ToPgxRows()
	successFunc := func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
		return expectedRow
	}

	ctxWithTx := postgres.ContextWithTx(context.Background(), s.tx)

	// With Tx
	s.mockTx.EXPECT().QueryRow(gomock.Any(), "command1").Times(1).DoAndReturn(successFunc)
	rows := s.conn.QueryRow(ctxWithTx, "command1")
	s.Equal(expectedRow, rows.Row)

	// Without Tx
	s.mockPgxPoolConn.EXPECT().QueryRow(gomock.Any(), "command2").Times(1).DoAndReturn(successFunc)
	rows = s.conn.QueryRow(context.Background(), "command2")
	s.Equal(expectedRow, rows.Row)
}

func (s *postgresConnTestSuite) TestConnSendBatch() {
	expectedResults := pkgmocks.NewMockBatchResults(s.mockCtrl)
	successFunc := func(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
		return expectedResults
	}

	ctxWithTx := postgres.ContextWithTx(context.Background(), s.tx)
	batch := &pgx.Batch{}

	// With Tx
	s.mockTx.EXPECT().SendBatch(gomock.Any(), batch).Times(1).DoAndReturn(successFunc)
	results := s.conn.SendBatch(ctxWithTx, batch)
	s.Same(expectedResults, results.BatchResults)

	// Without Tx
	s.mockPgxPoolConn.EXPECT().SendBatch(gomock.Any(), batch).Times(1).DoAndReturn(successFunc)
	results = s.conn.SendBatch(context.Background(), batch)
	s.Same(expectedResults, results.BatchResults)
}

func (s *postgresConnTestSuite) TestConnCopyFrom() {
	errFunc := func(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
		return 0, errFake
	}
	successFunc := func(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
		return 19, nil
	}

	ctxWithTx := postgres.ContextWithTx(context.Background(), s.tx)

	// With Tx
	id1 := pgx.Identifier{"foo"}
	s.mockTx.EXPECT().CopyFrom(gomock.Any(), id1, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(successFunc)
	n, err := s.conn.CopyFrom(ctxWithTx, id1, nil, nil)
	s.NoError(err)
	s.Equal(int64(19), n)

	// Without Tx
	id2 := pgx.Identifier{"bar"}
	s.mockPgxPoolConn.EXPECT().CopyFrom(gomock.Any(), id2, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(successFunc)
	n, err = s.conn.CopyFrom(context.Background(), id2, nil, nil)
	s.NoError(err)
	s.Equal(int64(19), n)

	// Error Handling with Tx
	id3 := pgx.Identifier{"foo", "bar"}
	s.mockTx.EXPECT().CopyFrom(gomock.Any(), id3, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(errFunc)
	n, err = s.conn.CopyFrom(ctxWithTx, id3, nil, nil)
	s.Equal(errFake, err)
	s.Zero(n)

	// Error Handling without Tx
	id4 := pgx.Identifier{"bar", "foo"}
	s.mockPgxPoolConn.EXPECT().CopyFrom(gomock.Any(), id4, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(errFunc)
	n, err = s.conn.CopyFrom(context.Background(), id4, nil, nil)
	s.Equal(errFake, err)
	s.Zero(n)
}
