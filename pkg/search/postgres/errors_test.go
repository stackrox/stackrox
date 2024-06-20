package postgres

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func Test_errExecQuery(t *testing.T) {
	const queryStr = "DROP TABLE USERS"
	err := errors.New("table USERS is too heavy to drop")
	err = errExecQuery(err, queryStr)

	assert.Equal(t, "error executing query", err.Error())
	assert.Equal(t, "error executing query DROP TABLE USERS: table USERS is too heavy to drop",
		errox.UnconcealSensitive(err))
}

func Test_errDeleteQuery(t *testing.T) {
	const queryStr = "DELETE * FROM USERS"
	err := errors.New("no such table")
	err = errDeleteQuery(err, "USERS", queryStr)

	assert.Equal(t, "could not delete from the database", err.Error())
	assert.Equal(t, "could not delete from \"USERS\" with query DELETE * FROM USERS: no such table",
		errox.UnconcealSensitive(err))
}
