package central

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	openshiftPkg "bitbucket.org/stack-rox/apollo/pkg/openshift"
	"bitbucket.org/stack-rox/apollo/pkg/zip"
)

func init() {
	Deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct {
	deploy         *template.Template
	clairifyCmd    *template.Template
	clairifyDeploy *template.Template
	cmd            *template.Template
	portForward    *template.Template
	rbac           *template.Template
}

func newOpenshift() deployer {
	return &openshift{
		deploy:         template.Must(template.New("openshift").Parse(k8sDeploy)),
		clairifyCmd:    template.Must(template.New("openshift").Parse(openshiftClairifyCmd)),
		clairifyDeploy: template.Must(template.New("openshift").Parse(openshiftClairifyYAML)),
		cmd:            template.Must(template.New("openshift").Parse(openshiftCmd)),
		portForward:    template.Must(template.New("openshift").Parse(getPortForwardTemplate("oc"))),
		rbac:           template.Must(template.New("openshift").Parse(openshiftCentralRBAC)),
	}
}

func (o *openshift) Render(c Config) ([]*v1.File, error) {
	var files []*v1.File
	data, err := executeTemplate(o.deploy, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.yaml", data, false))

	data, err = executeTemplate(o.cmd, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.sh", data, true))

	data, err = executeTemplate(o.rbac, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("rbac.yaml", data, false))

	data, err = executeTemplate(o.portForward, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("port-forward.sh", data, true))
	files = append(files, zip.NewFile("image-setup.sh", openshiftPkg.ImageSetup, true))

	data, err = executeTemplate(o.clairifyCmd, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("clairify.sh", data, true))

	data, err = executeTemplate(o.clairifyDeploy, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("clairify.yaml", data, false))
	return files, nil
}

var (
	openshiftCmd = commandPrefix + `
OC_PROJECT="{{.K8sConfig.Namespace}}"
OC_NAMESPACE="{{.K8sConfig.Namespace}}"
OC_SA="${OC_SA:-central}"

oc get project "${OC_PROJECT}" || oc new-project "${OC_PROJECT}"

echo "Adding cluster roles to the service account..."
oc create -f "${DIR}/rbac.yaml"
oc adm policy add-scc-to-user central "system:serviceaccount:${OC_PROJECT}:${OC_SA}"

oc create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
oc create secret -n "{{.K8sConfig.Namespace}}" generic central-jwt --from-file="$DIR/jwt-key.der"
oc create -f "$DIR/deploy.yaml"
`

	openshiftCentralRBAC = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: central
  namespace: {{.K8sConfig.Namespace}}
---
kind: SecurityContextConstraints
apiVersion: v1
metadata:
  annotations:
    kubernetes.io/description: central is the security constraint for the central server
  name: central
priority: 100
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
seccompProfiles:
- '*'
volumes:
- '*'
{{if .HostPath -}}
allowHostDirVolumePlugin: true
{{- end}}
`

	openshiftClairifyYAML = k8sClairifyYAML + k8sSeparator + openshiftClairifyRBAC

	openshiftClairifyRBAC = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: clairify
  namespace: {{.K8sConfig.Namespace}}
---
kind: SecurityContextConstraints
apiVersion: security.openshift.io/v1
metadata:
  annotations:
    kubernetes.io/description: clairify is the security constraint for the Clairify container
  name: clairify
priority: 100
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
seccompProfiles:
- '*'
volumes:
- '*'
`

	openshiftClairifyCmd = commandPrefix + `
OC_PROJECT={{.K8sConfig.Namespace}}
OC_NAMESPACE={{.K8sConfig.Namespace}}
OC_SA="${OC_SA:-clairify}"

oc create -f "$DIR/clairify.yaml"
oc adm policy add-scc-to-user clairify "system:serviceaccount:$OC_PROJECT:$OC_SA"
`
)
