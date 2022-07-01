package fixtures

import "github.com/stackrox/rox/generated/storage"

var (
	category = &storage.PolicyCategory{
		Id:        "category-id",
		Name:      "Boo's Special Category",
		IsDefault: false,
	}
)

// GetPolicyCategory returns a mock category
func GetPolicyCategory() *storage.PolicyCategory {
	return category.Clone()
}
