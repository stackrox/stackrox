package check132

const interpretationText = `StackRox has visibility into the effects of
Kubernetes Network Policies on your deployments. Network Policies can restrict
inbound and outbound traffic. Therefore, if every deployment has inbound
(ingress) Network Policies that apply to it, it can be considered compliant, so
long as none of those deployments are using the host namespace, which allows
circumvention of Network Policies.`
