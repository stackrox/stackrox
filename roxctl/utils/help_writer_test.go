package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_helpWriter(t *testing.T) {
	t.Run("empty line separator", func(t *testing.T) {
		sb := &strings.Builder{}
		w := makeHelpWriter(makeFormattingWriter(sb, 20))

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
		w := makeHelpWriter(makeFormattingWriter(sb, 25))
		w.WriteLn("short line of text")
		w.WriteLn("somewhat long line of text")
		w.Indent(2, 3).Write("<-2 spaces")
		w.WriteLn("\t<-\\t\\n->", "<-three spaces")
		w.WriteLn("no indent")
		assert.Equal(t,
			"short line of text\nsomewhat long line of \ntext\n  <-2 spaces\t<-\\t\\n->\n   <-three spaces\nno indent\n",
			sb.String())
	})
}
