package printers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPrintingObject struct {
	TestData testDataObject `json:"data"`
}

type testDataObject struct {
	TestAddresses []testAddressObject `json:"addresses"`
}

type testAddressObject struct {
	Name    string `json:"name"`
	ZIP     string `json:"zip"`
	Country string `json:"country"`
}

func TestTablePrinter_PrintWithMockData(t *testing.T) {
	columnHeaders := []string{
		"NAME", "ZIP", "COUNTRY",
	}
	columnExpressions := "{data.addresses.#.name,data.addresses.#.zip,data.addresses.#.country}"
	jsonObject := &testPrintingObject{
		TestData: testDataObject{
			TestAddresses: []testAddressObject{
				{
					Name:    "Test",
					ZIP:     "12345",
					Country: "Fictional1",
				},
				{
					Name:    "Test",
					ZIP:     "3456",
					Country: "Fictional1",
				},
				{
					Name:    "Test1",
					ZIP:     "3456",
					Country: "Fictional2",
				},

				{
					Name:    "Test1",
					ZIP:     "63438",
					Country: "Fictional4",
				},
			},
		},
	}

	cases := map[string]struct {
		expectedOutput string
		merge          bool
		noHeader       bool
	}{
		"table output without merging duplicate cells & rows": {
			merge:    false,
			noHeader: false,
			expectedOutput: `+-------+-------+------------+
| NAME  |  ZIP  |  COUNTRY   |
+-------+-------+------------+
| Test  | 12345 | Fictional1 |
+-------+-------+------------+
| Test  | 3456  | Fictional1 |
+-------+-------+------------+
| Test1 | 3456  | Fictional2 |
+-------+-------+------------+
| Test1 | 63438 | Fictional4 |
+-------+-------+------------+
`,
		},
		"table output with merging duplicate cells & rows": {
			merge:    true,
			noHeader: false,
			expectedOutput: `+-------+-------+------------+
| NAME  |  ZIP  |  COUNTRY   |
+-------+-------+------------+
| Test  | 12345 | Fictional1 |
+       +-------+            +
|       | 3456  |            |
+-------+       +------------+
| Test1 |       | Fictional2 |
+       +-------+------------+
|       | 63438 | Fictional4 |
+-------+-------+------------+
`,
		},
		"table output without merging duplicate cells & rows without headers": {
			merge:    false,
			noHeader: true,
			expectedOutput: `+-------+-------+------------+
| Test  | 12345 | Fictional1 |
+-------+-------+------------+
| Test  | 3456  | Fictional1 |
+-------+-------+------------+
| Test1 | 3456  | Fictional2 |
+-------+-------+------------+
| Test1 | 63438 | Fictional4 |
+-------+-------+------------+
`,
		},
		"table output with merging duplicate cells & rows without headers": {
			merge:    true,
			noHeader: true,
			expectedOutput: `+-------+-------+------------+
| Test  | 12345 | Fictional1 |
+       +-------+            +
|       | 3456  |            |
+-------+       +------------+
| Test1 |       | Fictional2 |
+       +-------+------------+
|       | 63438 | Fictional4 |
+-------+-------+------------+
`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := strings.Builder{}
			var columnsToMerge []string
			if c.merge {
				columnsToMerge = columnHeaders
			}
			printer := NewTablePrinter(columnExpressions, WithTableHeadersOption(columnHeaders, columnsToMerge, c.noHeader))
			require.NoError(t, printer.Print(&jsonObject, &out))
			assert.Equal(t, c.expectedOutput, out.String())
		})
	}
}
