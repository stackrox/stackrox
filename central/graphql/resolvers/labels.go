package resolvers

import (
	"sort"
)

type labelResolver struct {
	key, value string
}

func (r *labelResolver) Key() string {
	return r.key
}

func (r *labelResolver) Value() string {
	return r.value
}

type labels []*labelResolver

func (l labels) Len() int {
	return len(l)
}

func (l labels) Less(i, j int) bool {
	return l[i].key < l[j].key
}

func (l labels) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func labelsResolver(input map[string]string) labels {
	out := make(labels, len(input))
	i := 0
	for k, v := range input {
		out[i] = &labelResolver{k, v}
		i++
	}
	sort.Sort(out)
	return out
}
