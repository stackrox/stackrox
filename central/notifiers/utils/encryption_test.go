package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyChainParser(t *testing.T) {
	keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
	data := []byte(keyChainYaml)
	expected := &KeyChain{
		KeyMap: map[int]string{
			0: "key1",
			1: "key2",
			2: "key3",
		},
		ActiveKeyIndex: 2,
	}
	keyChain, err := parseKeyChainBytes(data)
	assert.NoError(t, err)
	assert.Equal(t, expected, keyChain)
}
