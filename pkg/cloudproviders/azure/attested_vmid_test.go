package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedCNs(t *testing.T) {
	t.Parallel()

	allowedCNs := []string{
		"metadata.azure.com",
		"msihostidentity.metadata.azure.com",
		"metadata.azure.us",
		"msihostidentity.metadata.azure.us",
		"metadata.azure.cn",
		"msihostidentity.metadata.azure.cn",
		"metadata.microsoftazure.de",
		"msihostidentity.metadata.microsoftazure.de",
	}

	disallowedCNs := []string{
		"azure.com",
		"xmetadata.azure.com",
		"azure.com.tv",
		"metadata.azure.com.tv",
		"foo.metadata.azure.com.tv",
	}

	for _, cn := range allowedCNs {
		assert.True(t, allowedCNsRegExp.MatchString(cn))
	}

	for _, cn := range disallowedCNs {
		assert.False(t, allowedCNsRegExp.MatchString(cn))
	}
}
