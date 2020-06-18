package printer

// A Func prints violation messages given a map of required fields to values.
type Func func(map[string][]string) ([]string, error)
