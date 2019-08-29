package preflight

var (
	preflightCheckList = []check{
		resourcesCheck{},
		schemaValidationCheck{},
		namespaceCheck{},
		labelsCheck{},
		objectPreconditionsCheck{},
		accessCheck{},
	}
)
