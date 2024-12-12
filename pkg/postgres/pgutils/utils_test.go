package pgutils

import (
	"net"
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

func TestNilOrCIDR(t *testing.T) {
	assert.Equal(t, &net.IPNet{
		IP:   net.IP{0x0a, 0x01, 0x02, 0x00},
		Mask: net.IPMask{0xff, 0xff, 0xff, 0x00},
	}, NilOrCIDR("10.1.2.0/24"))

	assert.Equal(t, (*net.IPNet)(nil), NilOrCIDR("invalid"))
	assert.Equal(t, (*net.IPNet)(nil), NilOrCIDR(""))
}
