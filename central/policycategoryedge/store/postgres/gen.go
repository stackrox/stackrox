package postgres

//go:generate pg-table-bindings-wrapper --get-all-func --type=storage.PolicyCategoryEdge --table=policy_category_edges --search-category POLICY_CATEGORY_EDGE --references=policies:storage.Policy,policy_categories:storage.PolicyCategory --search-scope POLICY_CATEGORY_EDGE,POLICY_CATEGORIES
