package standards

func buildQualifiedID(standardID, controlOrCategoryID string) string {
	return standardID + ":" + controlOrCategoryID
}
