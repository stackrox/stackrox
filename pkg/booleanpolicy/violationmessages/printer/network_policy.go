package printer

const (
	missingIngressNetworkPolicy = `Missing Ingress Network Policy violation message placeholder`
	missingEgressNetworkPolicy  = `Missing Egress Network Policy violation message placeholder`
)

// TODO(ROX-9760): Implement these functions according to UX decision on how to display violations for missing network policies.
// This is implemented with place-holder messages for now just to unblock further developments on the evaluation of
// this policy.

func missingIngressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	return executeTemplate(missingIngressNetworkPolicy, nil)
}

func missingEgressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	return executeTemplate(missingEgressNetworkPolicy, nil)
}
