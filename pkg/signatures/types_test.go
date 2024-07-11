package signatures

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteMetaSignatureIntegrationIDPrefix(t *testing.T) {
	test := regexp.QuoteMeta(SignatureIntegrationIDPrefix)
	assert.Equal(t, `io\.stackrox\.signatureintegration\.`, test)
}
