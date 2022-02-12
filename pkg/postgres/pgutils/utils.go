package pgutils

import (
	"reflect"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
)

// ErrNilIfNoRows returns nil if the error is pgx.ErrNoRows
func ErrNilIfNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

func ConvertEnumSliceToIntArray(i interface{}) []int32 {
	enumSlice := reflect.ValueOf(i)
	enumSliceLen := enumSlice.Len()
	resultSlice := make([]int32, 0, enumSliceLen)
	for i := 0; i < enumSlice.Len(); i++ {
		resultSlice = append(resultSlice, int32(enumSlice.Index(i).Int()))
	}
	return resultSlice
}

func NilOrStringTimestamp(t *types.Timestamp) *string {
	if t == nil {
		return nil
	}
	s := t.String()
	return &s
}
