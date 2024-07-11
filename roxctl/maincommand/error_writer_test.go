package maincommand

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

func TestErrorWriter(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "\nError: rpc error: code = Unauthenticated desc =\n credentials not found\n",
			out: "ERROR:\t\nError: rpc error: code = Unauthenticated desc =\n credentials not found\n",
		},
		{
			in:  "rpc error: code = Unauthenticated desc = credentials not found",
			out: "ERROR:\trpc error: code = Unauthenticated desc = credentials not found\n",
		},
		{
			in:  "rpc error: code = Unauthenticated desc = credentials not found\n",
			out: "ERROR:\trpc error: code = Unauthenticated desc = credentials not found\n",
		},
		{
			in:  "Error: rpc error: code = Unauthenticated desc = credentials not found",
			out: "ERROR:\trpc error: code = Unauthenticated desc = credentials not found\n",
		},
		{
			in:  "",
			out: "ERROR:\t\n",
		},
		{
			in:  "%s",
			out: "ERROR:\t%s\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			io, _, out, errorOut := io.TestIO()
			ew := errorWriter{
				logger: logger.NewLogger(io, printer.DefaultColorPrinter()),
			}
			n, err := ew.Write([]byte(tt.in))
			assert.NoError(t, err)
			assert.Len(t, tt.in, n)
			assert.Empty(t, out.String())
			assert.Equal(t, tt.out, errorOut.String())
		})
	}
}
