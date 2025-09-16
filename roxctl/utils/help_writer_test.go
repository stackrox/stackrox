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
		assert.NoError(t, w.err)
		assert.Equal(t, "word1\n\nword2\n", sb.String())
	})
	t.Run("write", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeHelpWriter(makeFormattingWriter(sb, 25, defaultTabWidth))
		w.WriteLn("short line of text")
		w.WriteLn("somewhat long line of text")
		w.Indent(2, 3).Write("<-2 sps")
		w.WriteLn("\t<-\\t\\n->", "<-three spaces")
		w.WriteLn("no indent")
		assert.NoError(t, w.err)
		assert.Equal(t, "short line of text\n"+
			"somewhat long line of \n"+
			"text\n"+
			"  <-2 sps\t<-\\t\\n->\n"+
			"   <-three spaces\n"+
			"no indent\n",
			sb.String())
	})
}
