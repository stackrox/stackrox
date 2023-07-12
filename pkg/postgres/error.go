package postgres

import (
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

func toErrox(err error) error {
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return errors.Wrap(errox.AlreadyExists, err.Error())
		}
	}
	return err
}
