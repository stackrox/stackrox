package globaldb

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numberOfClusters = 200

const sqlTemplate = `SELECT count(*) FROM test WHERE
	c_id = 0
 {{- range $i, $c := $.C }}
	OR (c_id = {{$c.ID}} {{if $c.N}} AND n_id IN (
		{{- range $index, $n := $c.N}}{{if $index}},{{end}}{{$n}}{{end}}){{ end -}})
 {{- end }};
`

const sqlTempTableTemplate = `DROP TABLE IF EXISTS eas;
CREATE TEMPORARY TABLE eas (c_id INT NOT NULL, n_id INT NOT NULL);
INSERT INTO eas (c_id, n_id) VALUES
(0, 0),
 {{- range $i, $c := $.C }}
	{{if $c.N}}{{- range $index, $n := $c.N}}({{$c.ID}}, {{$n}}),{{end}}{{ end -}}
 {{- end }}(0, 0);
`

type C struct {
	ID int
	N  []int
}

type filter struct {
	C []C
}

func BenchmarkTestPG(b *testing.B) {
	source := "host=localhost port=5431 database=postgres user=postgres password=Pass2020! sslmode=disable statement_timeout=6000000 pool_min_conns=90 pool_max_conns=90"
	pgInitialize(source)
	conn, err := pgDB.Acquire(context.Background())
	require.NoError(b, err)

	tag, err := conn.Exec(context.Background(), `DROP TABLE IF EXISTS eas;`)
	require.NoError(b, err)
	createTableSql := `
	CREATE TABLE IF NOT EXISTS eas (
	                    id SERIAL UNIQUE NOT NULL PRIMARY KEY,
	                    c_id INT NOT NULL,
	                    n_id INT NOT NULL,
	                    role INT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS  eas_index ON eas (c_id, n_id);
	CREATE INDEX IF NOT EXISTS  eas_c_index ON eas (c_id);
	CREATE INDEX IF NOT EXISTS  eas_name_index ON eas (role);`
	_, err = conn.Exec(context.Background(), createTableSql)
	require.NoError(b, err)
	println(tag.String())

	var numbers []int
	for i := 0; i <= numberOfClusters; i += 5 {
		numbers = append(numbers, i)
	}
	for _, numberOfClusters := range numbers {
		filter := prepareIDs(numberOfClusters)
		for _, c := range filter.C {
			for _, n := range c.N {
				_, err := conn.Exec(context.Background(), "INSERT INTO eas (c_id, n_id, role) VALUES ($1, $2, $3)", c.ID, n, numberOfClusters)
				require.NoError(b, err)
			}
		}
	}
	println("Generated role table")
	for _, numberOfClusters := range numbers {
		// prerun -- generate query and get expected count
		query := fmt.Sprintf(`SELECT count(*) FROM test INNER JOIN eas ON (eas.c_id = test.c_id AND eas.n_id = test.n_id) WHERE eas.role = %d`, numberOfClusters)
		require.NoError(b, err)
		count := execute(b, conn, query)
		b.Run(fmt.Sprintf("eas role single numberOfClusters=%d count=%d query_len=%d", numberOfClusters, count, len(query)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				execute(b, conn, query)
			}
		})
	}

	for _, numberOfClusters := range numbers {
		// prerun -- generate query and get expected count
		query := fmt.Sprintf(`SELECT count(*) FROM test INNER JOIN eas ON (eas.c_id = test.c_id AND eas.n_id = test.n_id) WHERE eas.role IN (%d,%d,%d)`, numberOfClusters-2, numberOfClusters-1, numberOfClusters)
		require.NoError(b, err)
		count := execute(b, conn, query)
		b.Run(fmt.Sprintf("eas role 3 numberOfClusters=%d count=%d query_len=%d", numberOfClusters, count, len(query)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				execute(b, conn, query)
			}
		})
	}

	q := "SELECT count(*) FROM test"
	count := execute(b, conn, q)
	b.Run(fmt.Sprintf(fmt.Sprintf("%s, count=%d", q, count)), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			execute(b, conn, q)
		}
	})

	for _, numberOfClusters := range numbers {
		// prerun -- generate query and get expected count
		query, err := generateQuery(prepareIDs(numberOfClusters), sqlTemplate)
		require.NoError(b, err)
		count := execute(b, conn, query)
		b.Run(fmt.Sprintf("where numberOfClusters=%d count=%d query_len=%d", numberOfClusters, count, len(query)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				execute(b, conn, query)
			}
		})
	}

	for _, numberOfClusters := range numbers {
		// prerun -- generate query and get expected count
		eas, err := generateQuery(prepareIDs(numberOfClusters), sqlTempTableTemplate)
		query := `SELECT count(*) FROM test INNER JOIN eas ON (eas.c_id = test.c_id AND eas.n_id = test.n_id)`
		require.NoError(b, err)
		tag, err := conn.Exec(context.Background(), eas)
		require.NoError(b, err)
		println(tag.String())
		count := execute(b, conn, query)
		b.Run(fmt.Sprintf("tmp table numberOfClusters=%d count=%d query_len=%d", numberOfClusters, count, len(query)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = conn.Exec(context.Background(), eas)
				require.NoError(b, err)
				execute(b, conn, query)
			}
		})
	}
}

func prepareIDs(numberOfClusters int) filter {
	f := filter{
		C: make([]C, 0, numberOfClusters),
	}
	for i := 0; i < numberOfClusters; i++ {
		nids := make([]int, 0, i)
		for j := 0; j < i; j++ {
			nids = append(nids, j)
		}
		f.C = append(f.C, C{ID: i, N: nids})
	}
	return f
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

func generateQuery(filter filter, sqlTemplate string) (string, error) {
	tmpl := template.Must(template.New("tmpl").Funcs(template.FuncMap{
		"IntsJoin": intsJoin,
	}).Parse(sqlTemplate))
	var query bytes.Buffer
	err := tmpl.Execute(&query, filter)
	return query.String(), err
}

func intsJoin(ints []int) string {
	str := make([]string, 0, len(ints))
	for _, v := range ints {
		str = append(str, strconv.Itoa(v))
	}
	return strings.Join(str, ", ")
}
