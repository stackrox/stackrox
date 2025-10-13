package types

import "strings"

// Digest is a wrapper around a SHA so we can access it with or without a prefix
type Digest struct {
	algorithm string
	hash      string
}

// NewDigest returns an internal representation of a SHA.
// If an algorithm cannot be determined, it is set to sha256
// for legacy purposes.
func NewDigest(sha string) *Digest {
	if sha == "" {
		return nil
	}
	var hash, algorithm string
	if idx := strings.Index(sha, ":"); idx != -1 {
		algorithm = sha[:idx]
		hash = sha[idx+1:]
	} else {
		algorithm = "sha256"
		hash = sha
	}
	return &Digest{
		algorithm: algorithm,
		hash:      hash,
	}
}

// Algorithm returns the algorithm used in the Digest
func (d *Digest) Algorithm() string {
	if d == nil {
		return ""
	}
	return d.algorithm
}

// Digest returns the entire Digest
func (d *Digest) Digest() string {
	if d == nil {
		return ""
	}
	return d.algorithm + ":" + d.hash
}

// Hash returns the SHA without the sha256: prefix.
func (d *Digest) Hash() string {
	if d == nil {
		return ""
	}
	return d.hash
}

func (d *Digest) String() string {
	return d.Digest()
}
