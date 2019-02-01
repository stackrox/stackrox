package check112

const interpretationText = `StackRox generates a network diagram based on
observed network communication for each deployment. Therefore, because every
deployment is contained in the network graph, StackRox provides compliance.`

func passText() string {
	return "StackRox shows all connections between deployments as well as connections from deployments to outside of the cluster."
}
