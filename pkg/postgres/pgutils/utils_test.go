package pgutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgxpoolDsnToPgxDsn(t *testing.T) {
	assert.Equal(t, "host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000",
		PgxpoolDsnToPgxDsn("host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90"))
	assert.Equal(t, "host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000",
		PgxpoolDsnToPgxDsn("pool_min_conns=1 host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000 pool_max_conns=90"))
	assert.Equal(t, "host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000",
		PgxpoolDsnToPgxDsn(" host=localhost port=5432 database=postgres user=who pool_min_conns=1 password=password sslmode=disable statement_timeout=600000 pool_max_conns=90"))
}
