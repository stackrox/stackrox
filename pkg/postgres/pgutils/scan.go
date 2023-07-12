package pgutils

import (
	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v5"
)

type unmarshaler[T any] interface {
	proto.Unmarshaler
	*T
}

// ScanRows scan and unmarshal postgres rows into object of type T.
func ScanRows[T any, PT unmarshaler[T]](rows pgx.Rows) ([]*T, error) {
	var results []*T
	for rows.Next() {
		msg, err := unmarshal[T, PT](rows)
		if err != nil {
			return nil, err
		}
		results = append(results, msg)
	}
	return results, rows.Err()
}

func unmarshal[T any, PT unmarshaler[T]](row pgx.Row) (*T, error) {
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, err
	}
	msg := new(T)
	if err := PT(msg).Unmarshal(data); err != nil {
		return nil, err
	}
	return msg, nil
}
