package helpers

func ConvertEnumArray[IN any, OUT any](input []IN, convert func(IN) OUT) []OUT {
	if input == nil {
		return nil
	}
	result := make([]OUT, len(input))
	for i, in := range input {
		result[i] = convert(in)
	}
	return result
}

func ConvertPointerArray[IN any, OUT any](input []*IN, convert func(*IN) *OUT) []*OUT {
	if input == nil {
		return nil
	}
	result := make([]*OUT, 0, len(input))
	for _, in := range input {
		if in == nil {
			continue
		}
		result = append(result, convert(in))
	}
	return result
}
