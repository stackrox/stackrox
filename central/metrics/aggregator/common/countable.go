package common

type Countable interface{ Count() int }

// OneOrMore is a helper implementation of Countable interface, that counts 1
// by default. Can be used as a base for other implementations.
type OneOrMore int

func (o OneOrMore) Count() int {
	if o > 0 {
		return int(o)
	}
	return 1
}
