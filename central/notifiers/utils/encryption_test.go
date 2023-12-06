package utils

import (
	"testing"

	"github.com/pkg/errors"
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

func TestGetActiveNotifierEncryptionKey(t *testing.T) {
	// case: successful reading keychain
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
		return []byte(keyChainYaml), nil
	}
	key, idx, err := GetActiveNotifierEncryptionKey()
	assert.NoError(t, err)
	assert.Equal(t, "key3", key)
	assert.Equal(t, 2, idx)

	// case: error reading file
	keyChainFileReader = func(_ string) ([]byte, error) {
		return nil, errors.New("file not found")
	}
	_, _, err = GetActiveNotifierEncryptionKey()
	assert.Error(t, err)

	// case: active index does not exist
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 100
`
		return []byte(keyChainYaml), nil
	}
	_, _, err = GetActiveNotifierEncryptionKey()
	assert.Error(t, err)
}

func TestGetNotifierEncryptionKeyAtIndex(t *testing.T) {
	// case: successful reading keychain
	keyChainFileReader = func(_ string) ([]byte, error) {
		keyChainYaml := `
keyMap:
  0: key1
  1: key2
  2: key3
activeKeyIndex: 2
`
		return []byte(keyChainYaml), nil
	}
	key, err := GetNotifierEncryptionKeyAtIndex(1)
	assert.NoError(t, err)
	assert.Equal(t, "key2", key)

	// case: index does not exist
	_, err = GetNotifierEncryptionKeyAtIndex(100)
	assert.Error(t, err)

	// case: error reading file
	keyChainFileReader = func(_ string) ([]byte, error) {
		return nil, errors.New("user does not have read permission")
	}
	_, err = GetNotifierEncryptionKeyAtIndex(1)
	assert.Error(t, err)
}
