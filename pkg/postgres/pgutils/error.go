package pgutils

import (
	"io"
	"net"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/net/context"
)

var transientPGCodes = set.NewFrozenStringSet(
	// Class 08 — Connection Exception
	"08000", // connection_exception
	"08003", // connection_does_not_exist
	"08006", // connection_failure
	"08001", // sqlclient_unable_to_establish_sqlconnection
	"08004", // sqlserver_rejected_establishment_of_sqlconnection
	"08007", // transaction_resolution_unknown
	"08P01", // protocol_violation

	// Class 40 — Transaction Rollback
	"40000", // transaction_rollback
	"40002", // transaction_integrity_constraint_violation
	"40001", // serialization_failure
	"40003", // statement_completion_unknown
	"40P01", // deadlock_detected

	// Class 55 — Object Not In Prerequisite State
	"55000", // object_not_in_prerequisite_state
	"55006", // object_in_use
	"55P03", // lock_not_available

	// Class 57 — Operator Intervention
	"57000", // operator_intervention
	"57014", // query_canceled
	"57P01", // admin_shutdown
	"57P02", // crash_shutdown
	"57P03", // cannot_connect_now
	"57P05", // idle_session_timeout

	// Class 58 — System Error (errors external to PostgreSQL itself)
	"58000", // system_error
	"58030", // io_error
)

// IsTransientError specifies if the passed error is transient and should be retried
func IsTransientError(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	if multiError := (*errorhelpers.ErrorList)(nil); errors.As(err, &multiError) {
		for _, err := range multiError.Errors() {
			if IsTransientError(err) {
				return true
			}
		}
	}
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
		return transientPGCodes.Contains(pgErr.Code)
	}
	if pgconn.SafeToRetry(err) {
		return true
	}
	if errorhelpers.IsAny(err, pgx.ErrNoRows, pgx.ErrTxClosed, pgx.ErrTxCommitRollback) {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if errorhelpers.IsAny(err, context.DeadlineExceeded) {
		return true
	}
	if errorhelpers.IsAny(err, io.EOF, io.ErrUnexpectedEOF, io.ErrClosedPipe, syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED, syscall.EPIPE) {
		return true
	}
	if err := errors.Unwrap(err); err != nil {
		return IsTransientError(err)
	}
	return false
}

const (
	errCodeUniqueConstraint = "23505"
)

// IsUniqueConstraintError specifies if the passed error is due to a unique constraint violation.
func IsUniqueConstraintError(err error) bool {
	if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) {
		return pgErr.Code == errCodeUniqueConstraint
	}
	return false
}
