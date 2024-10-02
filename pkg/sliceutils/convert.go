package sliceutils

// ConvertSlice is a generic function to convert slices of any type
func ConvertSlice[T any, U any](input []T, convertFunc func(T) U) []U {
	if input == nil {
		return nil
	}
	output := make([]U, 0, len(input))
	for _, v := range input {
		output = append(output, convertFunc(v))
	}
	return output
}
