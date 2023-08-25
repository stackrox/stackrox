package loaders

func collectMissing(ids []string, missing []int) []string {
	missingIds := make([]string, 0, len(missing))
	for _, missingIdx := range missing {
		missingIds = append(missingIds, ids[missingIdx])
	}
	return missingIds
}
