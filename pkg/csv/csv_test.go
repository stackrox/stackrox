package csv

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	type row struct {
		a string
		b string
	}

	rows := []*row{{
		a: "a one",
		b: "b one",
	}, {
		a: "a two",
		b: "b two",
	}}
	buf := bytes.NewBuffer(nil)
	w := NewStreamWriter[row](buf,
		WithHeader("a", "b"),
		WithBOM(),
		WithCRLF(),
		WithConverter(func(r *row) ([]string, error) {
			return []string{r.a, r.b}, nil
		}))
	var err error
	for _, m := range rows {
		if err = w.AddRow(m); err != nil {
			break
		}
	}
	err = w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, string(utf8BOM)+"a,b\r\na one,b one\r\na two,b two\r\n", buf.String())
}

func TestFlushHeaders(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewStreamWriter[struct{}](buf, WithHeader("a", "b"))
	err := w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, "a,b\n", buf.String())
}

func TestReflectedHeaderAndConverter(t *testing.T) {
	type row struct {
		A string
		B string `csv:"X"`
	}

	buf := bytes.NewBuffer(nil)
	w := NewStreamWriter[row](buf)
	err := w.AddRow(&row{
		A: "a one",
		B: "b one",
	})
	assert.NoError(t, err)
	err = w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, "A,X\na one,b one\n", buf.String())
}

func TestErrors(t *testing.T) {
	type row struct {
		a string
		b string
	}

	rows := []*row{{
		a: "a one",
		b: "b one",
	}, {
		a: "a two",
		b: "b two",
	}}
	errBadRow := errors.New("bad row")
	buf := bytes.NewBuffer(nil)
	w := NewStreamWriter[row](buf,
		WithNoHeader(),
		WithConverter(func(r *row) ([]string, error) {
			return nil, errBadRow
		}))

	var err error
	for _, m := range rows {
		if err = w.AddRow(m); err != nil {
			break
		}
	}
	assert.ErrorIs(t, err, errBadRow)

	err = w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())
}
