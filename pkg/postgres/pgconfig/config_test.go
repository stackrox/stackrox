package pgconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSource(t *testing.T) {
	tests := map[string]struct {
		input       string
		expected    map[string]string
		expectError bool
	}{
		"basic connection string": {
			input: "host=localhost port=5432 user=postgres dbname=test",
			expected: map[string]string{
				"host":   "localhost",
				"port":   "5432",
				"user":   "postgres",
				"dbname": "test",
			},
		},
		"connection string with password": {
			input: "host=localhost port=5432 user=postgres password=secret dbname=test",
			expected: map[string]string{
				"host":     "localhost",
				"port":     "5432",
				"user":     "postgres",
				"password": "secret",
				"dbname":   "test",
			},
		},
		"password with equals sign": {
			input: "host=localhost port=5432 user=postgres password=secret=value dbname=test",
			expected: map[string]string{
				"host":     "localhost",
				"port":     "5432",
				"user":     "postgres",
				"password": "secret=value",
				"dbname":   "test",
			},
		},
		"connection string with sslmode and timeouts": {
			input: "host=central-db.stackrox port=5432 user=postgres sslmode=verify-full statement_timeout=600000",
			expected: map[string]string{
				"host":              "central-db.stackrox",
				"port":              "5432",
				"user":              "postgres",
				"sslmode":           "verify-full",
				"statement_timeout": "600000",
			},
		},
		"connection string with pool parameters": {
			input: "host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90",
			expected: map[string]string{
				"host":              "localhost",
				"port":              "5432",
				"database":          "postgres",
				"user":              "who",
				"password":          "password",
				"sslmode":           "disable",
				"statement_timeout": "600000",
				"pool_min_conns":    "1",
				"pool_max_conns":    "90",
			},
		},
		"connection string with extra whitespace": {
			input: "  host=localhost   port=5432   user=postgres  ",
			expected: map[string]string{
				"host": "localhost",
				"port": "5432",
				"user": "postgres",
			},
		},
		"connection string with client_encoding": {
			input: "host=localhost port=5432 user=postgres client_encoding=UTF8",
			expected: map[string]string{
				"host":            "localhost",
				"port":            "5432",
				"user":            "postgres",
				"client_encoding": "UTF8",
			},
		},
		"value with trailing spaces gets trimmed": {
			input: "host=localhost   port=5432  ",
			expected: map[string]string{
				"host": "localhost",
				"port": "5432",
			},
		},
		"empty source string": {
			input:       "",
			expectError: true,
		},
		"key without value": {
			input: "host=localhost port",
			expected: map[string]string{
				"host": "localhost",
				"port": "",
			},
		},
		"multiple equals in value": {
			input: "host=localhost password=a=b=c",
			expected: map[string]string{
				"host":     "localhost",
				"password": "a=b=c",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ParseSource(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func FuzzParseSource(f *testing.F) {
	// Seed with valid PostgreSQL connection string examples
	f.Add("host=localhost port=5432 user=postgres dbname=test")
	f.Add("host=localhost port=5432 user=postgres password=secret dbname=test")
	f.Add("host=central-db.stackrox port=5432 user=postgres sslmode=verify-full statement_timeout=600000")
	f.Add("host=localhost port=5432 database=postgres user=who password=password sslmode=disable statement_timeout=600000 pool_min_conns=1 pool_max_conns=90")
	f.Add("user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10")
	f.Add("host=testHost port=5432 database=testDB sensitiveField=testSensitive")
	f.Add("host=127.0.0.1 port=5432 user=postgres sslmode=disable")

	// Edge cases and special characters
	f.Add("password=has=equals=signs host=localhost")
	f.Add("  host=localhost   port=5432  ")
	f.Add("host= localhost  port= 5432 ")
	f.Add("key=value")
	f.Add("a=b c=d")

	// Invalid/empty cases
	f.Add("")
	f.Add(" ")
	f.Add("=")
	f.Add("==")
	f.Add("key")
	f.Add("key=")
	f.Add("=value")

	// Special characters
	f.Add("password=p@ssw0rd! host=localhost")
	f.Add("host=localhost password=with spaces")
	f.Add("host=localhost user=user@domain.com")

	// Unicode and special encodings
	f.Add("host=localhost password=パスワード")
	f.Add("host=localhost dbname=тест")

	// Long values
	f.Add("host=localhost password=verylongpasswordverylongpasswordverylongpassword")
	f.Add("host=very.long.hostname.example.com.with.many.dots.and.subdomains.test.local")

	f.Fuzz(func(_ *testing.T, source string) {
		result, _ := ParseSource(source)
		for k, v := range result {
			_ = k
			_ = v
		}
	})
}
