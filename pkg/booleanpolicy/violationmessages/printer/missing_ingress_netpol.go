package printer

const (
	missingIngres = `MISSING INGRESS!!`
)

func missingIngressPrinter(fieldMap map[string][]string) ([]string, error) {
	return executeTemplate(missingIngres, nil)
}
