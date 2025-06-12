package common

import "slices"

type Expression []*Condition

func (expr Expression) match(value string) bool {
	for _, cond := range expr {
		if !cond.match(value) {
			return false
		}
	}
	return true
}

func (expr Expression) Equals(another Expression) bool {
	return slices.EqualFunc(expr, another, (*Condition).Equals)
}
