package central

import (
	"fmt"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/zip"
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct {
	deploy      *template.Template
	clairify    *template.Template
	cmd         *template.Template
	lb          *template.Template
	portForward *template.Template
}

func newKubernetes() deployer {
	return &kubernetes{
		deploy:      template.Must(template.New("kubernetes").Parse(k8sDeploy)),
		clairify:    template.Must(template.New("kubernetes").Parse(k8sClairifyYAML)),
		cmd:         template.Must(template.New("kubernetes").Parse(k8sCmd)),
		lb:          template.Must(template.New("kubernetes").Parse(k8sLB)),
		portForward: template.Must(template.New("kubernetes").Parse(getPortForwardTemplate("kubectl"))),
	}
}

func (k *kubernetes) Render(c Config) ([]*v1.File, error) {
	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.PreventImage)
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

	data, err = executeTemplate(k.clairify, c)
	if err != nil {
		return nil, err
	}
	files = append(files, zip.NewFile("clairify.yaml", data, false))
	files = append(files, zip.NewFile("clairify.sh", k8sClairifyScript, true))

	return files, nil
}

func getPortForwardTemplate(cmd string) string {
	return fmt.Sprintf(k8sPortForwardTemplate, cmd, cmd, cmd)
}

var (
	k8sSeparator = `
---
`

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
  minReadySeconds: 15
  selector:
    matchLabels:
      app: central
  template:
    metadata:
      namespace: {{.K8sConfig.Namespace}}
      labels:
        app: central
    spec:
      {{if .HostPath -}}
      nodeSelector:
        {{.HostPath.NodeSelectorKey}}: {{.HostPath.NodeSelectorValue}}
      {{- end}}
      {{if eq .ClusterType.String "KUBERNETES_CLUSTER" }}
      imagePullSecrets:
      - name: {{.K8sConfig.ImagePullSecret}}
      {{else}}
      serviceAccount: central
      {{- end}}
      containers:
      - name: central
        image: {{.K8sConfig.PreventImage}}
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "8Gi"
            cpu: "2000m"
        command:
        - central
        ports:
        - name: api
          containerPort: 443
        securityContext:
          capabilities:
            drop: ["NET_RAW"]
        volumeMounts:
        - name: central-certs-volume
          mountPath: /run/secrets/stackrox.io/certs/
          readOnly: true
        - name: central-jwt-volume
          mountPath: /run/secrets/stackrox.io/jwt/
          readOnly: true
        {{if .HostPath -}}
        - name: {{.HostPath.Name}}
          mountPath: {{.HostPath.MountPath}}
        {{- end}}
        {{if .External -}}
        - name: {{.External.Name}}
          mountPath: {{.External.MountPath}}
        {{- end}}
      volumes:
      - name: central-certs-volume
        secret:
          secretName: central-tls
      - name: central-jwt-volume
        secret:
          secretName: central-jwt
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
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-ext-to-central
  namespace: {{.K8sConfig.Namespace}}
spec:
  ingress:
  - ports:
    - port: 443
      protocol: TCP
  podSelector:
    matchLabels:
      app: central
  policyTypes:
  - Ingress
`

	k8sCmd = commandPrefix + kubernetesPkg.GetCreateSecretTemplate("{{.K8sConfig.Namespace}}", "{{.K8sConfig.Registry}}", "{{.K8sConfig.ImagePullSecret}}") + `
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
kubectl create secret -n "{{.K8sConfig.Namespace}}" generic central-jwt --from-file="$DIR/jwt-key.der"
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
if [[ -z "$1" ]]; then
	echo "usage: bash port-forward.sh 8000"
	echo "The above would forward localhost:8000 to central:443."
	exit 1
fi

until [ "$(%s get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' | grep Running | wc -l)" -eq 1 ]; do
    echo -n .
    sleep 1
done

export CENTRAL_POD="$(%s get pod -n {{.K8sConfig.Namespace}} --selector 'app=central' --output=jsonpath='{.items..metadata.name} {.items..status.phase}' | grep Running | cut -f 1 -d ' ')"
%s port-forward -n "{{.K8sConfig.Namespace}}" "${CENTRAL_POD}" "$1:443" > /dev/null &
echo "Access central on localhost:$1"
`

	k8sClairifyYAML = `
apiVersion: v1
kind: Service
metadata:
  name: clairify
  namespace: {{.K8sConfig.Namespace}}
spec:
  ports:
  - name: clair-http
    port: 6060
    targetPort: 6060
  - name: clairify-http
    port: 8080
    targetPort: 8080
  selector:
    app: clairify
  type: ClusterIP
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: clairify
  namespace: {{.K8sConfig.Namespace}}
  labels:
    app: clairify
  annotations:
    owner: stackrox
    email: support@stackrox.com
spec:
  replicas: 1
  minReadySeconds: 15
  selector:
    matchLabels:
      app: clairify
  template:
    metadata:
      namespace: {{.K8sConfig.Namespace}}
      labels:
        app: clairify
    spec:
      containers:
      - name: clairify
        image: {{.K8sConfig.ClairifyImage}}
        resources:
          requests:
            memory: "500Mi"
            cpu: "500m"
          limits:
            memory: "2000Mi"
            cpu: "2000m"
        env:
        - name: CLAIR_ARGS
          value: "-insecure-tls"
        command:
          - /init
          - /clairify
        imagePullPolicy: Always
        ports:
        - name: clair
          containerPort: 6060
        - name: clairify
          containerPort: 8080
        securityContext:
          capabilities:
            drop: ["NET_RAW"]
      {{if eq .ClusterType.String "KUBERNETES_CLUSTER" }}
      imagePullSecrets:
      - name: {{.K8sConfig.ImagePullSecret}}
      {{else}}
      serviceAccount: clairify
      {{- end}}
`

	k8sClairifyScript = commandPrefix + `

kubectl create -f "${DIR}/clairify.yaml"
`
)
