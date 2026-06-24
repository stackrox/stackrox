package uuid

// Standalone SHA-1 (FIPS 180-4) that does not import crypto/sha1.
// UUID v5 (RFC 4122) mandates SHA-1 for deterministic ID derivation,
// but Go's fips140=only mode blanket-rejects crypto/sha1 even for
// non-cryptographic use. This implementation is invisible to that check.

import (
	"encoding/binary"
	"hash"
	"math/bits"
)

const (
	sha1Size      = 20
	sha1BlockSize = 64
)

type sha1digest struct {
	h   [5]uint32
	x   [sha1BlockSize]byte
	nx  int
	len uint64
}

func newSHA1() hash.Hash {
	d := new(sha1digest)
	d.Reset()
	return d
}

func (d *sha1digest) Reset() {
	d.h[0] = 0x67452301
	d.h[1] = 0xEFCDAB89
	d.h[2] = 0x98BADCFE
	d.h[3] = 0x10325476
	d.h[4] = 0xC3D2E1F0
	d.nx = 0
	d.len = 0
}

func (d *sha1digest) Size() int      { return sha1Size }
func (d *sha1digest) BlockSize() int { return sha1BlockSize }

func (d *sha1digest) Write(p []byte) (int, error) {
	nn := len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == sha1BlockSize {
			sha1Block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	for len(p) >= sha1BlockSize {
		sha1Block(d, p[:sha1BlockSize])
		p = p[sha1BlockSize:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return nn, nil
}

func (d *sha1digest) Sum(in []byte) []byte {
	d0 := *d
	hash := d0.checkSum()
	return append(in, hash[:]...)
}

func (d *sha1digest) checkSum() [sha1Size]byte {
	len := d.len
	var tmp [sha1BlockSize + 8]byte
	tmp[0] = 0x80
	var padLen int
	if len%sha1BlockSize < 56 {
		padLen = int(56 - len%sha1BlockSize)
	} else {
		padLen = int(64 + 56 - len%sha1BlockSize)
	}
	binary.BigEndian.PutUint64(tmp[padLen:], len*8)
	d.Write(tmp[:padLen+8]) //nolint:errcheck

	var digest [sha1Size]byte
	binary.BigEndian.PutUint32(digest[0:], d.h[0])
	binary.BigEndian.PutUint32(digest[4:], d.h[1])
	binary.BigEndian.PutUint32(digest[8:], d.h[2])
	binary.BigEndian.PutUint32(digest[12:], d.h[3])
	binary.BigEndian.PutUint32(digest[16:], d.h[4])
	return digest
}

func sha1Block(d *sha1digest, p []byte) {
	var w [80]uint32
	for i := 0; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(p[i*4:])
	}
	for i := 16; i < 80; i++ {
		w[i] = bits.RotateLeft32(w[i-3]^w[i-8]^w[i-14]^w[i-16], 1)
	}

	a, b, c, d0, e := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4]

	for i := 0; i < 80; i++ {
		var f, k uint32
		switch {
		case i < 20:
			f = (b & c) | (^b & d0)
			k = 0x5A827999
		case i < 40:
			f = b ^ c ^ d0
			k = 0x6ED9EBA1
		case i < 60:
			f = (b & c) | (b & d0) | (c & d0)
			k = 0x8F1BBCDC
		default:
			f = b ^ c ^ d0
			k = 0xCA62C1D6
		}
		t := bits.RotateLeft32(a, 5) + f + e + k + w[i]
		e = d0
		d0 = c
		c = bits.RotateLeft32(b, 30)
		b = a
		a = t
	}

	d.h[0] += a
	d.h[1] += b
	d.h[2] += c
	d.h[3] += d0
	d.h[4] += e
}
