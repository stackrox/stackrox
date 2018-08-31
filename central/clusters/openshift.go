package clusters

import (
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	openshiftPkg "github.com/stackrox/rox/pkg/openshift"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newOpenshift()
}

type openshift struct {
	deploy *template.Template
	cmd    *template.Template
	rbac   *template.Template
	delete *template.Template
}

func newOpenshift() Deployer {
	return &openshift{
		deploy: template.Must(template.New("openshift").Parse(k8sDeploy)),
		cmd:    template.Must(template.New("openshift").Parse(openshiftCmd)),
		rbac:   template.Must(template.New("openshift").Parse(openshiftRBAC)),
		delete: template.Must(template.New("openshift").Parse(openshiftDelete)),
	}
}

func (o *openshift) Render(c Wrap) ([]*v1.File, error) {
	var openshiftParams *v1.OpenshiftParams
	clusterOpenshift, ok := c.OrchestratorParams.(*v1.Cluster_Openshift)
	if ok {
		openshiftParams = clusterOpenshift.Openshift
	}

	fields := fieldsFromWrap(c)
	addCommonKubernetesParams(openshiftParams.GetParams(), fields)
	fields["OpenshiftAPIEnv"] = env.OpenshiftAPI.EnvVar()
	fields["OpenshiftAPI"] = `"true"`

	var files []*v1.File
	data, err := executeTemplate(o.deploy, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.yaml", data, false))

	data, err = executeTemplate(o.cmd, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.sh", data, true))

	data, err = executeTemplate(o.rbac, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("rbac.yaml", data, false))

	data, err = executeTemplate(o.delete, fields)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("delete.sh", data, true))

	files = append(files, zip.NewFile("image-setup.sh", openshiftPkg.ImageSetup, true))

	return files, nil
}

var (
	openshiftCmd = commandPrefix + `

OC_PROJECT={{.Namespace}}
OC_NAMESPACE={{.Namespace}}
OC_SA="${OC_SA:-sensor}"
OC_BENCHMARK_SA="${OC_BENCHMARK_SA:-benchmark}"

oc create -f "$DIR/rbac.yaml"

oc adm policy add-scc-to-user sensor "system:serviceaccount:$OC_PROJECT:$OC_SA"
oc adm policy add-scc-to-user benchmark "system:serviceaccount:$OC_PROJECT:$OC_BENCHMARK_SA"

oc create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/central-ca.pem"
oc create -f "$DIR/deploy.yaml"
`

	openshiftRBAC = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sensor
  namespace: {{.Namespace}}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: benchmark
  namespace: {{.Namespace}}
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:monitor-deployments
rules:
  - resources:
    - daemonsets
    - deployments
    - deploymentconfigs
    - pods
    - replicasets
    - replicationcontrollers
    - services
    - statefulsets
    - secrets
    apiGroups:
    - '*'
    verbs:
    - get
    - watch
    - list
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:monitor-deployments-binding
subjects:
- kind: ServiceAccount
  name: sensor
  namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: {{.Namespace}}:monitor-deployments
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:enforce-policies
rules:
  - resources:
    - daemonsets
    - deployments
    - deploymentconfigs
    - pods
    - replicasets
    - replicationcontrollers
    - services
    - statefulsets
    apiGroups:
    - '*'
    verbs:
    - update
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:enforce-policies-binding
subjects:
- kind: ServiceAccount
  name: sensor
  namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: {{.Namespace}}:enforce-policies
  apiGroup: rbac.authorization.k8s.io
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:launch-benchmarks
  namespace: {{.Namespace}}
rules:
  - resources:
    - daemonsets
    apiGroups:
    - extensions
    verbs:
    - create
    - get
    - delete
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.Namespace}}:launch-benchmarks-binding
  namespace: {{.Namespace}}
subjects:
- kind: ServiceAccount
  name: sensor
  namespace: {{.Namespace}}
roleRef:
  kind: Role
  name: {{.Namespace}}:launch-benchmarks
  apiGroup: rbac.authorization.k8s.io
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: {{.Namespace}}:network-policies
rules:
  - resources:
    - networkpolicies
    apiGroups:
    - '*'
    verbs:
    - get
    - watch
    - list
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: {{.Namespace}}:network-policies-binding
subjects:
- kind: ServiceAccount
  name: sensor
  namespace: {{.Namespace}}
roleRef:
  kind: ClusterRole
  name: {{.Namespace}}:network-policies
  apiGroup: rbac.authorization.k8s.io
---
kind: SecurityContextConstraints
apiVersion: v1
metadata:
  annotations:
    kubernetes.io/description: sensor is the security constraint for the sensor server
  name: sensor
priority: 100
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
seccompProfiles:
- '*'
---
kind: SecurityContextConstraints
apiVersion: security.openshift.io/v1
metadata:
  annotations:
    kubernetes.io/description: benchmark is the security constraint for the benchmark container
  name: benchmark
priority: 100
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
seccompProfiles:
- '*'
allowHostDirVolumePlugin: true
allowHostPID: true
volumes:
- '*'
`

	openshiftDelete = commandPrefix + `
	oc delete -f "$DIR/deploy.yaml"
	oc delete -n {{.Namespace}} secret/sensor-tls
`
)
