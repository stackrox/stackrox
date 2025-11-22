package postgres

//go:generate pg-table-bindings-wrapper --type=storage.PolicyCategory --cached-store --search-category POLICY_CATEGORIES --default-sort search.PolicyCategoryName.String() --transform-sort-options PolicyCategoriesSchema.OptionsMap
