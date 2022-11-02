package nvd

import (
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

type Matcher interface {
	Matches(cve *schema.NVDCVEFeedJSON10DefCVEItem) bool
}

type cpeMatcher struct {
	substring string
}

func NewCPEMatcher(substring string) Matcher {
	return cpeMatcher{
		substring: substring,
	}
}

func (m cpeMatcher) Matches(entry *schema.NVDCVEFeedJSON10DefCVEItem) bool {
	for _, node := range entry.Configurations.Nodes {
		for _, cpeMatch := range node.CPEMatch {
			if strings.Contains(cpeMatch.Cpe23Uri, m.substring) {
				return true
			}
		}
	}
	return false
}
