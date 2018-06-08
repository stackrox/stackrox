package central

import (
	"fmt"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	kubernetesPkg "bitbucket.org/stack-rox/apollo/pkg/kubernetes"
	"bitbucket.org/stack-rox/apollo/pkg/zip"
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct {
	deploy      *template.Template
	cmd         *template.Template
	lb          *template.Template
	portForward *template.Template
}

func newKubernetes() deployer {
	return &kubernetes{
		deploy:      template.Must(template.New("kubernetes").Parse(k8sDeploy)),
		cmd:         template.Must(template.New("kubernetes").Parse(k8sCmd)),
		lb:          template.Must(template.New("kubernetes").Parse(k8sLB)),
		portForward: template.Must(template.New("kubernetes").Parse(getPortForwardTemplate("kubectl"))),
	}
}

func (k *kubernetes) Render(c Config) ([]*v1.File, error) {
	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.Image)
	if err != nil {
		return nil, err
	}
	var files []*v1.File
	data, err := executeTemplate(k.deploy, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.yaml", data, false))

	data, err = executeTemplate(k.cmd, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("deploy.sh", data, true))

	data, err = executeTemplate(k.lb, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("lb.yaml", data, false))

	data, err = executeTemplate(k.portForward, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("port-forward.sh", data, true))

	return files, nil
}

func getPortForwardTemplate(cmd string) string {
	return fmt.Sprintf(k8sPortForwardTemplate, cmd, cmd, cmd)
}

var (
	k8sDeploy = `apiVersion: v1
kind: Service
metadata:
  name: central
  namespace: {{.K8sConfig.Namespace}}
spec:
  ports:
  - name: https
    port: 443
    targetPort: 443
  selector:
    app: central
  type: ClusterIP
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: central
  namespace: {{.K8sConfig.Namespace}}
  labels:
    app: central
  annotations:
    owner: stackrox
    email: support@stackrox.com
spec:
  replicas: 1
  selector:
    matchLabels:
      app: central
  template:
    metadata:
      namespace: stackrox
      labels:
        app: central
    spec:
      {{if .HostPath -}}
      nodeSelector:
        {{.HostPath.NodeSelectorKey}}: {{.HostPath.NodeSelectorValue}}
      {{- end}}
      containers:
      - name: central
        image: {{.K8sConfig.Image}}
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "8Gi"
            cpu: "2000m"
        imagePullPolicy: Always
        command:
        - central
        ports:
        - name: api
          containerPort: 443
        securityContext:
          capabilities:
            drop: ["NET_RAW"]
        volumeMounts:
        - name: certs
          mountPath: /run/secrets/stackrox.io/
          readOnly: true
        {{if .HostPath -}}
        - name: {{.HostPath.Name}}
          mountPath: {{.HostPath.MountPath}}
        {{- end}}
        {{if .External -}}
        - name: {{.External.Name}}
          mountPath: {{.External.MountPath}}
        {{- end}}
      imagePullSecrets:
      - name: {{.K8sConfig.ImagePullSecret}}
      volumes:
      - name: certs
        secret:
          secretName: central-tls
      {{if .HostPath -}}
      - name: {{.HostPath.Name}}
        hostPath:
          path: {{.HostPath.HostPath}}
          type: Directory
      {{- end}}
      {{if .External -}}
      - name: {{.External.Name}}
        persistentVolumeClaim:
          claimName: {{.External.Name}}
      {{- end}}
{{if .External -}}
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: {{.External.Name}}
  namespace: {{.K8sConfig.Namespace}}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
{{- end}}
`

	k8sCmd = commandPrefix + kubernetesPkg.GetCreateSecretTemplate("{{.K8sConfig.Namespace}}", "{{.K8sConfig.Registry}}", "{{.K8sConfig.ImagePullSecret}}") + `
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
kubectl create -f "${DIR}/deploy.yaml"
echo "Central has been deployed"
`

	k8sLB = `apiVersion: v1
kind: Service
metadata:
  name: central-lb
  namespace: {{.K8sConfig.Namespace}}
spec:
  ports:
  - port: 443
    targetPort: 443
  selector:
    app: central
  type: LoadBalancer
`

	k8sPortForwardTemplate = commandPrefix + `
if [[ -z $1 ]]; then
	echo "usage: bash port-forward.sh 8000"
	echo "The above would forward localhost:8000 to central:443."
	exit 1
fi

until [ "$(%s get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' | grep Running | wc -l)" -eq 1 ]; do
    echo -n .
    sleep 1
done

export CENTRAL_POD="$(%s get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
%s port-forward -n "{{.K8sConfig.Namespace}}" "${CENTRAL_POD}" $1:443 > /dev/null &
echo "Access central on localhost:$1"

`
)
