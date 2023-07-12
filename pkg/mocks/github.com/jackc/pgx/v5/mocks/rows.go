package mocks

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// This file was copied and fixed
// https://github.com/driftprogramming/pgxpoolmock/blob/acbdb300842addabf2ff2df80d107592cb9455e1/rows.go

type rowSets struct {
	sets []*Rows
	pos  int
}

func (rs *rowSets) Err() error {
	r := rs.sets[rs.pos]
	return r.nextErr[r.pos-1]
}

func (rs *rowSets) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("")
}

func (rs *rowSets) FieldDescriptions() []pgconn.FieldDescription {
	return rs.sets[rs.pos].defs
}

func (rs *rowSets) Close() {}

// Next advances to next row
func (rs *rowSets) Next() bool {
	r := rs.sets[rs.pos]
	r.pos++
	return r.pos <= len(r.rows)
}

func (rs *rowSets) Values() ([]interface{}, error) {
	return nil, nil
}

func (rs *rowSets) Scan(dest ...interface{}) error {
	r := rs.sets[rs.pos]
	if len(dest) != len(r.defs) {
		return fmt.Errorf("incorrect argument number %d for columns %d", len(dest), len(r.defs))
	}
	for i, col := range r.rows[r.pos] {
		if dest[i] == nil {
			// behave compatible with pgx
			continue
		}
		destVal := reflect.ValueOf(dest[i])
		if destVal.Kind() != reflect.Ptr {
			return fmt.Errorf("destination argument must be a pointer for column %s", r.defs[i].Name)
		}
		if col == nil {
			dest[i] = nil
			continue
		}
		val := reflect.ValueOf(col)
		if destVal.Elem().Kind() == val.Kind() {
			if destElem := destVal.Elem(); destElem.CanSet() {
				destElem.Set(val)
			} else {
				return fmt.Errorf("cannot set destination value for column %s", r.defs[i].Name)
			}
		} else {
			return fmt.Errorf("destination kind '%v' not supported for value kind '%v' of column '%s'",
				destVal.Elem().Kind(), val.Kind(), r.defs[i].Name)
		}
	}
	return r.nextErr[r.pos-1]
}

func (rs *rowSets) RawValues() [][]byte {
	r := rs.sets[rs.pos]
	dest := make([][]byte, len(r.defs))

	for i, col := range r.rows[r.pos-1] {
		if b, ok := rawBytes(col); ok {
			dest[i] = b
			continue
		}
		dest[i] = col.([]byte)
	}

	return dest
}

// transforms to debuggable printable string
func (rs *rowSets) String() string {
	if rs.empty() {
		return "with empty rows"
	}

	msg := "should return rows:\n"
	if len(rs.sets) == 1 {
		for n, row := range rs.sets[0].rows {
			msg += fmt.Sprintf("    row %d - %+v\n", n, row)
		}
		return strings.TrimSpace(msg)
	}
	for i, set := range rs.sets {
		msg += fmt.Sprintf("    result set: %d\n", i)
		for n, row := range set.rows {
			msg += fmt.Sprintf("      row %d - %+v\n", n, row)
		}
	}
	return strings.TrimSpace(msg)
}

func (rs *rowSets) empty() bool {
	for _, set := range rs.sets {
		if len(set.rows) > 0 {
			return false
		}
	}
	return true
}

func rawBytes(col interface{}) (_ []byte, ok bool) {
	val, ok := col.([]byte)
	if !ok || len(val) == 0 {
		return nil, false
	}
	// Copy the bytes from the mocked row into a shared raw buffer, which we'll replace the content of later
	// This allows scanning into sql.RawBytes to correctly become invalid on subsequent calls to Next(), Scan() or Close()
	b := make([]byte, len(val))
	copy(b, val)
	return b, true
}

// Rows is a mocked collection of rows to
// return for Query result
type Rows struct {
	defs     []pgconn.FieldDescription
	rows     [][]interface{}
	pos      int
	nextErr  map[int]error
	closeErr error
}

// NewRows allows Rows to be created from a
// sql interface{} slice or from the CSV string and
// to be used as sql driver.Rows.
// Use Sqlmock.NewRows instead if using a custom converter
func NewRows(columns []string) *Rows {
	var coldefs []pgconn.FieldDescription
	for _, column := range columns {
		coldefs = append(coldefs, pgconn.FieldDescription{Name: column})
	}
	return &Rows{
		defs:    coldefs,
		nextErr: make(map[int]error),
	}
}

// CloseError allows to set an error
// which will be returned by rows.Close
// function.
//
// The close error will be triggered only in cases
// when rows.Next() EOF was not yet reached, that is
// a default sql library behavior
func (r *Rows) CloseError(err error) *Rows {
	r.closeErr = err
	return r
}

// RowError allows to set an error
// which will be returned when a given
// row number is read
func (r *Rows) RowError(row int, err error) *Rows {
	r.nextErr[row] = err
	return r
}

// AddRow composed of database interface{} slice
// return the same instance to perform subsequent actions.
// Note that the number of values must match the number
// of columns
func (r *Rows) AddRow(values ...interface{}) *Rows {
	if len(values) != len(r.defs) {
		panic("Expected number of values to match number of columns")
	}

	row := make([]interface{}, len(r.defs))
	copy(row, values)
	r.rows = append(r.rows, row)
	return r
}

// ToPgxRows convert Rows into pgx.Rows that could be used as a mocked responses.
func (r *Rows) ToPgxRows() pgx.Rows {
	pgxRows := convert(r)
	return pgxRows
}

// HasNextResultSet implement the "RowsNextResultSet" interface
func (rs *rowSets) HasNextResultSet() bool {
	return rs.pos+1 < len(rs.sets)
}

// NextResultSet implements the "RowsNextResultSet" interface
func (rs *rowSets) NextResultSet() error {
	if !rs.HasNextResultSet() {
		return io.EOF
	}

	rs.pos++
	return nil
}

// Conn implements the pgx.Row interface
func (rs *rowSets) Conn() *pgx.Conn {
	return nil
}

func convert(rows ...*Rows) pgx.Rows {
	defs := 0
	sets := make([]*Rows, len(rows))
	for i, r := range rows {
		sets[i] = r
		if r.defs != nil {
			defs++
		}
	}
	return &rowSets{sets: sets}
}
