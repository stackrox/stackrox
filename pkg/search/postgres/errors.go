package postgres

import "github.com/stackrox/rox/pkg/errox"

func errExecQuery(err error, queryStr string) error {
	return errox.NewSensitive(
		errox.WithPublicMessage("error executing query"),
		errox.WithSensitive(err),
		errox.WithSensitivef("error executing query %s", queryStr))
}

func errDeleteQuery(err error, table, queryStr string) error {
	return errox.NewSensitive(
		errox.WithPublicMessage("could not delete from the database"),
		errox.WithSensitive(err),
		errox.WithSensitivef("could not delete from %q with query %s", table, queryStr))
}
