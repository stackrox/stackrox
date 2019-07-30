package ioutils

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
