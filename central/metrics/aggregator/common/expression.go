package common

type Expression []*Condition

func (expr Expression) match(value string) bool {
	if len(expr) == 0 {
		return true
	}
	for _, group := range expr.splitByOR() {
		matched := true
		for _, cond := range group {
			if !cond.match(value) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func (expr Expression) splitByOR() []Expression {
	var groups []Expression
	current := []*Condition{}
	for _, cond := range expr {
		if cond.op == opOR {
			groups = append(groups, current)
			current = []*Condition{}
		} else {
			current = append(current, cond)
		}
	}
	return append(groups, current)
}
