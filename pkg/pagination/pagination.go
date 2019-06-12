package pagination

import v1 "github.com/stackrox/rox/generated/api/v1"

// CalculatePaginationIndices returns the indices that the slice should be filtered by
func CalculatePaginationIndices(length int, constMaxLength int, pagination *v1.Pagination) (start, end int) {
	if pagination == nil {
		if length > constMaxLength {
			return 0, constMaxLength
		}
		return 0, length
	}
	offset := int(pagination.GetOffset())
	if offset >= length {
		return 0, 0
	}
	limit := int(pagination.GetLimit())
	finalIndex := offset + limit
	if finalIndex >= length {
		finalIndex = length
	}
	return offset, finalIndex
}
