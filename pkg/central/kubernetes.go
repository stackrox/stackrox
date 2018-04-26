package central

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

func newKubernetes() deployer {
	return &basicDeployer{
		deploy: template.Must(template.New("kubernetes").Parse(k8sDeploy)),
		cmd:    template.Must(template.New("kubernetes").Parse(k8sCmd)),
	}
}

var (
	k8sDeploy = `apiVersion: v1
kind: Service
metadata:
  name: central
  namespace: {{.Namespace}}
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
  namespace: {{.Namespace}}
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
      containers:
      - name: central
        image: {{.Image}}
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
      imagePullSecrets:
      - name: stackrox
      volumes:
      - name: certs
        secret:
          secretName: central-tls
`

	k8sCmd = commandPrefix + `kubectl create secret -n "{{.Namespace}}" generic central-tls --from-file="$DIR/ca.pem" --from-file="$DIR/ca-key.pem"
kubectl create -f "$DIR/deploy.yaml"
`
)
