package compound

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

type searchRequestSpec struct {
	or      []*searchRequestSpec
	and     []*searchRequestSpec
	boolean *booleanRequestSpec

	base *baseRequestSpec
}

type booleanRequestSpec struct {
	must    *searchRequestSpec
	mustNot *searchRequestSpec
}

type baseRequestSpec struct {
	Spec  *SearcherSpec
	Query *v1.Query
}
