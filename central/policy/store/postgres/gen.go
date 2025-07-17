package postgres

//go:generate pg-table-bindings-wrapper --type=storage.Policy --search-category POLICIES --default-sort search.SORTPolicyName.String() --transform-sort-options PoliciesSchema.OptionsMap
