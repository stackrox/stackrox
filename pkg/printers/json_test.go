package printers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJSONObject struct {
	Name     string    `json:"name"`
	Street   string    `json:"street"`
	City     cityJSON  `json:"city"`
	Gender   string    `json:"gender"`
	Phone    phoneJSON `json:"phone"`
	HTMLChar string    `json:"htmlChar"`
}

type phoneJSON struct {
	Prefix string `json:"prefix"`
	Number string `json:"number"`
}
type cityJSON struct {
	Name    string `json:"name"`
	ZIPCode string `json:"zip"`
}

func TestJsonPrinter_Print(t *testing.T) {
	jsonObj := testJSONObject{
		Name:     "testname",
		Street:   "teststreet",
		HTMLChar: "something >=<& someone",
		City: cityJSON{
			Name:    "testcity",
			ZIPCode: "testZIP",
		},
		Gender: "test",
		Phone: phoneJSON{
			Prefix: "+1",
			Number: "123456789",
		},
	}

	cases := map[string]struct {
		compact        bool
		escapeHTML     bool
		expectedOutput string
	}{
		"Prettified JSON output unescaped": {
			expectedOutput: `{
  "name": "testname",
  "street": "teststreet",
  "city": {
    "name": "testcity",
    "zip": "testZIP"
  },
  "gender": "test",
  "phone": {
    "prefix": "+1",
    "number": "123456789"
  },
  "htmlChar": "something >=<& someone"
}
`,
		},
		"Prettified JSON output escaped": {
			escapeHTML: true,
			expectedOutput: `{
  "name": "testname",
  "street": "teststreet",
  "city": {
    "name": "testcity",
    "zip": "testZIP"
  },
  "gender": "test",
  "phone": {
    "prefix": "+1",
    "number": "123456789"
  },
  "htmlChar": "something \u003e=\u003c\u0026 someone"
}
`,
		},
		"Compact JSON output escaped": {
			compact:        true,
			escapeHTML:     true,
			expectedOutput: `{"name":"testname","street":"teststreet","city":{"name":"testcity","zip":"testZIP"},"gender":"test","phone":{"prefix":"+1","number":"123456789"},"htmlChar":"something \u003e=\u003c\u0026 someone"}`,
		},
		"Compact JSON output": {
			compact:        true,
			expectedOutput: `{"name":"testname","street":"teststreet","city":{"name":"testcity","zip":"testZIP"},"gender":"test","phone":{"prefix":"+1","number":"123456789"},"htmlChar":"something >=<& someone"}`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := strings.Builder{}
			printer := NewJSONPrinter(WithJSONCompact(c.compact), WithJSONEscapeHTML(c.escapeHTML))
			require.NoError(t, printer.Print(&jsonObj, &out))
			assert.Equal(t, c.expectedOutput, out.String())
		})
	}
}
