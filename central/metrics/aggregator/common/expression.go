package common

type Expression []*Condition

func (expr Expression) match(value string) bool {
	for _, cond := range expr {
		if !cond.match(value) {
			return false
		}
	}
	return true
}
