package postgres

import (
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

func toErrox(err error) error {
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
		// Ref: https://www.postgresql.org/docs/current/errcodes-appendix.html.
		switch pgErr.Code {
		case "23505":
			return errox.AlreadyExists.New(err.Error())
		case "23503":
			// Special case: for insert and update operations a FK constraint violation can occur when the referenced
			// FK does not exist. Instead of returning errox.ReferencedByAnotherObject, we shall return
			// errox.ReferencedObjectNotFound here.
			// The format of the detail message will be of:
			// Key (X)=(Y) is not present in table "Z".
			if strings.Contains(pgErr.Detail, "is not present in table") {
				return errox.ReferencedObjectNotFound.New(err.Error())
			}
			return errox.ReferencedByAnotherObject.New(err.Error())
		}
	}
	return err
}
