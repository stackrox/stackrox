package globaldb

import (
	"bytes"
	"context"
	"testing"
	"text/template"

	"github.com/cenkalti/backoff/v3"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const qTemplate = `SELECT count(*) FROM test WHERE n_id IN (0{{- range $val := Iterate . }},{{ $val }}{{- end }});`

func TestLongQuery(t *testing.T) {
	ctx := context.Background()
	source := "host=localhost port=5431 database=postgres user=postgres password=Pass2020! sslmode=disable statement_timeout=6000000 pool_min_conns=90 pool_max_conns=90"
	config, err := pgxpool.ParseConfig(source)
	require.NoError(t, err)
	pgDB, err := pgxpool.ConnectConfig(ctx, config)

	tmpl := template.Must(template.New("tmpl").Funcs(template.FuncMap{
		"Iterate": func(count int) []int {
			var i int
			var Items []int
			for i = 1; i <= (count); i++ {
				Items = append(Items, 0)
			}
			return Items
		},
	}).Parse(qTemplate))

	//i := sort.Search(7000000, func(i int) bool {
	//	println("i = ", i)
	//	if i < 6670108 {
	//		return false
	//	}
	//	var query bytes.Buffer
	//	err = tmpl.Execute(&query, i)
	//	require.NoError(t, err)
	//
	//	var rows pgx.Rows
	//	err = backoff.Retry(func() error {
	//		rows, err = pgDB.Query(context.Background(), query.String())
	//		if err != nil {
	//			println(err.Error())
	//		}
	//		return err
	//	}, backoff.NewExponentialBackOff())
	//	require.NoError(t, err)
	//	defer rows.Close()
	//	return !rows.Next()
	//})
	//println(i) // 6670111

	var query bytes.Buffer
	err = tmpl.Execute(&query, 6670111)
	require.NoError(t, err)

	var rows pgx.Rows
	err = backoff.Retry(func() error {
		rows, err = pgDB.Query(context.Background(), query.String())
		if err != nil {
			println(err.Error())
		}
		return err
	}, backoff.NewExponentialBackOff())
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())
	values, err := rows.Values()
	require.NoError(t, err)
	assert.Len(t, values, 1)
	assert.Equal(t, int64(0), values[0])
}
