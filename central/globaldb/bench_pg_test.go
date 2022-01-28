package globaldb

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numberOfIDs = 3000

const sqlTemplate = `SELECT count(*) FROM test WHERE
	c_id = 0
 {{- range $cid := $.C }}
	OR (c_id = {{$cid}} {{if $.N}} AND n_id IN (
		{{- range $index, $n := $.N}}{{if $index}},{{end}}{{$n}}{{end}}){{ end -}})
 {{- end }};
`

type filter struct {
	C, N []int
}

func BenchmarkTestPG(b *testing.B) {
	source := "host=localhost port=5432 database=postgres user=postgres sslmode=disable statement_timeout=600000 pool_min_conns=90 pool_max_conns=90"
	pgInitialize(source)
	conn, err := pgDB.Acquire(context.Background())
	require.NoError(b, err)

	cids, nids := prepareIDs()

	q := "SELECT count(*) FROM test"
	count := execute(b, conn, q)
	b.Run(fmt.Sprintf(fmt.Sprintf("%s, count=%d", q, count)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			execute(b, conn, q)
		}
	})

	for numberOfCids := 0; numberOfCids < numberOfIDs; numberOfCids = (numberOfCids + 1) * 2 {
		for numberOfNids := 0; numberOfNids < numberOfIDs; numberOfNids = (numberOfNids + 1) * 2 {
			// prerun -- generate query and get expected count
			query, err := generateQuery(cids[:numberOfCids], nids[:numberOfNids])
			require.NoError(b, err)
			count := execute(b, conn, query)
			b.Run(fmt.Sprintf("c=%d\tn=%d\tcount=%d", numberOfCids, numberOfNids, count), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					execute(b, conn, query)
				}
			})
		}
	}
}

func prepareIDs() ([]int, []int) {
	cids := make([]int, numberOfIDs)
	nids := make([]int, numberOfIDs)
	for i := 0; i < numberOfIDs; i++ {
		cids[i] = i
		nids[i] = i
	}

	rand.Shuffle(len(cids), func(i, j int) {
		cids[i], cids[j] = cids[j], cids[i]
	})
	rand.Shuffle(len(nids), func(i, j int) {
		nids[i], nids[j] = nids[j], nids[i]
	})
	return cids, nids
}

func execute(t *testing.B, conn *pgxpool.Conn, query string) int64 {
	rows, err := conn.Query(context.Background(), query)
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	values, err := rows.Values()
	require.NoError(t, err)
	assert.Len(t, values, 1)
	count := values[0].(int64)
	return count
}

func generateQuery(cids []int, nids []int) (string, error) {
	tmpl := template.Must(template.New("tmpl").Funcs(template.FuncMap{
		"IntsJoin": intsJoin,
	}).Parse(sqlTemplate))
	var query bytes.Buffer
	err := tmpl.Execute(&query, filter{
		C: cids,
		N: nids,
	})
	return query.String(), err
}

func intsJoin(ints []int) string {
	str := make([]string, 0, len(ints))
	for _, v := range ints {
		str = append(str, strconv.Itoa(v))
	}
	return strings.Join(str, ", ")
}
