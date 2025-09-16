package utils

// BatchProcess calls f with slices of the provided set. Slices are batchSize size max.
func BatchProcess[T interface{}](set []T, batchSize int, f func([]T) error) error {
	localBatchSize := batchSize
	for {
		if len(set) == 0 {
			break
		}

		if len(set) < localBatchSize {
			localBatchSize = len(set)
		}

		batch := set[:localBatchSize]
		if err := f(batch); err != nil {
			return err
		}

		set = set[localBatchSize:]
	}
	return nil
}
