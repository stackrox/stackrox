package language

import (
	"github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/stackrox/scanner/ext/versionfmt"
)

// ParserName is the name by which the language parser is registered.
const ParserName = "language"

type parser struct{}

func (p parser) Valid(v string) bool {
	panic("required function not implemented")
}

func (p parser) Compare(a, b string) (int, error) {
	return nvd.SmartVerCmp(a, b), nil
}

func (p parser) Namespaces() []string {
	return []string{"language"}
}

func init() {
	versionfmt.RegisterParser(ParserName, parser{})
}
