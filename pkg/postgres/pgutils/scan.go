package pgutils

import (
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Unmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler.
type Unmarshaler[T any] interface {
	UnmarshalVTUnsafe(dAtA []byte) error
	*T
}

// ScanStrings scans strings from rows.
//
// This function closes the rows automatically on return.
func ScanStrings(rows pgx.Rows) ([]string, error) {
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (string, error) {
		var str string
		if err := r.Scan(&str); err != nil {
			return "", errors.Wrap(err, "scanning string")
		}
		return str, nil
	})
}

// ScanRows scan and Unmarshal postgres rows into object of type T.
//
// This function closes the rows automatically on return.
func ScanRows[T any, PT Unmarshaler[T]](rows pgx.Rows) ([]*T, error) {
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (*T, error) {
		return Unmarshal[T, PT](r)
	})
}

// Unmarshal postgres row into object of type T
func Unmarshal[T any, PT Unmarshaler[T]](row pgx.Row) (*T, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}
	msg := new(T)
	if err := PT(msg).UnmarshalVTUnsafe(data); err != nil {
		errList := errorhelpers.NewErrorList("Unmarshalling error")
		errList.AddError(errors.Wrap(err, "Unmarshalling from bytes"))
		jsonUnmarshalErr := protojson.Unmarshal(data, interface{}(msg).(proto.Message))
		if jsonUnmarshalErr != nil {
			errList.AddError(errors.Wrap(jsonUnmarshalErr, "Unmarshalling from json"))
			return nil, errList
		}
	}
	return msg, nil
}
