package clusters

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func init() {
	deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

func newOpenshift() deployer {
	return &basicDeployer{
		deploy:    template.Must(template.New("openshift").Parse(k8sDeploy)),
		cmd:       template.Must(template.New("openshift").Parse(openshiftCmd)),
		addFields: addOpenShiftFields,
	}
}

func addOpenShiftFields(c Wrap, fields map[string]string) {
	addKubernetesFields(c, fields)

	fields["OpenshiftAPI"] = `"true"`
}

var openshiftCmd = commandPrefix + `oc create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/central-ca.pem"
oc create -f "$DIR/sensor-deploy.yaml"
`
