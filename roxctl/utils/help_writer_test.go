package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_helpWriter(t *testing.T) {
	t.Run("empty line separator", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeHelpWriter(makeFormattingWriter(sb, 20, defaultTabWidth))

		w.EmptyLineSeparator()
		w.EmptyLineSeparator()
		w.EmptyLineSeparator()
		w.WriteLn("word1")
		w.EmptyLineSeparator()
		w.EmptyLineSeparator()
		w.EmptyLineSeparator()
		w.WriteLn("word2")
		assert.Equal(t, "word1\n\nword2\n", sb.String())
	})
	t.Run("write", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeHelpWriter(makeFormattingWriter(sb, 25, defaultTabWidth))
		w.WriteLn("\nshort line of text")
		w.WriteLn("somewhat long line of text")
		w.Indent(2, 3).Write("<-2 sps")
		w.WriteLn("\t<-\\t\\n->", "<-three spaces")
		w.WriteLn("no indent")
		assert.Equal(t, `
short line of text
somewhat long line of 
text
  <-2 sps	<-\t\n->
   <-three spaces
no indent
`,
			sb.String())
	})
}
