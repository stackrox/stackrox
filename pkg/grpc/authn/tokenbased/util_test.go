package tokenbased

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestExtractBearerToken(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"Bearer foobar"},
	}

	token := ExtractToken(md, "bearer")
	assert.Equal(t, "foobar", token)
}

func TestExtractInvalidType(t *testing.T) {
	md := metadata.MD{
		"authorization": []string{"token foobar"},
	}

	token := ExtractToken(md, "Bearer")
	assert.Empty(t, token)
}
