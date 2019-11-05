package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLWithQueryParams(t *testing.T) {
	baseURL := "https://www.example.com"
	queryParams := []QueryParam{
		{
			Key:   "foo",
			Value: "foo",
		},
		{
			Key:   "bar",
			Value: "bar",
		},
	}
	url, err := GetURLWithQueryParams(baseURL, queryParams)
	assert.Nil(t, err)
	assert.Equal(t, url, "https://www.example.com?bar=bar&foo=foo")
}
