package clusters

import (
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/env"
)

func init() {
	deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

func newKubernetes() deployer {
	return &basicDeployer{
		deploy:    template.Must(template.New("kubernetes").Parse(k8sDeploy)),
		cmd:       template.Must(template.New("kubernetes").Parse(k8sCmd)),
		addFields: addKubernetesFields,
	}
}

func addKubernetesFields(c Wrap, fields map[string]string) {
	namespace := "default"
	if len(c.Namespace) != 0 {
		namespace = c.Namespace
	}
	fields["Namespace"] = namespace
	fields["ImagePullSecretEnv"] = env.ImagePullSecrets.EnvVar()
	fields["ImagePullSecret"] = c.ImagePullSecret
}

var (
	k8sDeploy = `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: sensor
  namespace: {{.Namespace}}
  labels:
    app: sensor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sensor
  template:
    metadata:
      namespace: {{.Namespace}}
      labels:
        app: sensor
    spec:
      containers:
      - image: {{.Image}}
        env:
        - name: {{.PublicEndpointEnv}}
          value: {{.PublicEndpoint}}
        - name: {{.ClusterIDEnv}}
          value: {{.ClusterID}}
        - name: {{.ImageEnv}}
          value: {{.Image}}
        - name: {{.AdvertisedEndpointEnv}}
          value: sensor.{{.Namespace}}:443
        - name: {{.ImagePullSecretEnv}}
          value: {{.ImagePullSecret}}
        - name: ROX_PREVENT_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: ROX_PREVENT_SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        imagePullPolicy: Always
        name: sensor
        command:
        - kubernetes-sensor
        volumeMounts:
        - name: certs
          mountPath: /run/secrets/stackrox.io/
          readOnly: true
      imagePullSecrets:
      - name: {{.ImagePullSecret}}
      volumes:
      - name: certs
        secret:
          secretName: sensor-tls
          items:
          - key: sensor-cert.pem
            path: cert.pem
          - key: sensor-key.pem
            path: key.pem
          - key: central-ca.pem
            path: ca.pem
---
apiVersion: v1
kind: Service
metadata:
  name: sensor
  namespace: {{.Namespace}}
spec:
  ports:
  - name: https
    port: 443
    targetPort: 443
  selector:
    app: sensor
  type: ClusterIP`

	k8sCmd = commandPrefix + `kubectl create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/central-ca.pem"
kubectl create -f "$DIR/sensor-deploy.yaml"
`
)
