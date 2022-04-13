package compound

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

type searchRequestSpec struct {
	or                     []*searchRequestSpec
	and                    []*searchRequestSpec
	boolean                *booleanRequestSpec
	leftJoinWithRightOrder *joinRequestSpec
	base                   *baseRequestSpec
}

type booleanRequestSpec struct {
	must    *searchRequestSpec
	mustNot *searchRequestSpec
}

type joinRequestSpec struct {
	left  *searchRequestSpec
	right *searchRequestSpec
}

type baseRequestSpec struct {
	Spec  *SearcherSpec
	Query *v1.Query
}
