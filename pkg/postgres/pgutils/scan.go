package pgutils

import (
	"github.com/jackc/pgx/v5"
)

// Unmarshaler is a generic interface type wrapping around types that implement protobuf Unmarshaler.
type Unmarshaler[T any] interface {
	UnmarshalVTUnsafe(dAtA []byte) error
	*T
}

// ScanRows scan and Unmarshal postgres rows into object of type T.
func ScanRows[T any, PT Unmarshaler[T]](rows pgx.Rows) ([]*T, error) {
	var results []*T
	for rows.Next() {
		msg, err := Unmarshal[T, PT](rows)
		if err != nil {
			return nil, err
		}
		results = append(results, msg)
	}
	return results, rows.Err()
}

// Unmarshal postgres row into object of type T
func Unmarshal[T any, PT Unmarshaler[T]](row pgx.Row) (*T, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}
	msg := new(T)
	if err := PT(msg).UnmarshalVTUnsafe(data); err != nil {
		return nil, err
	}
	return msg, nil
}
