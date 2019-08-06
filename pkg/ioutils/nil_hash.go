package ioutils

import (
	"github.com/pkg/errors"
)

type nilHash struct{}

func (nilHash) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func (nilHash) Sum(b []byte) []byte {
	return b
}

func (nilHash) Reset() {}

func (nilHash) Size() int { return 0 }

func (nilHash) BlockSize() int { return 1 }

func (nilHash) MarshalBinary() ([]byte, error) { return nil, nil }
func (nilHash) UnmarshalBinary(data []byte) error {
	if len(data) > 0 {
		return errors.Errorf("nil hash serialization should be empty, is %d bytes long", len(data))
	}
	return nil
}
