package postgres

import (
	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

func toErrox(err error) error {
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return errors.Wrap(errox.AlreadyExists, err.Error())
		case "23503":
			return errors.Wrap(errox.ReferencedByAnotherObject, err.Error())
		}
	}
	return err
}
