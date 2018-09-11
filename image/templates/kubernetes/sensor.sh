#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl get namespace {{.Namespace}} > /dev/null || kubectl create namespace {{.Namespace}}

if ! kubectl get secret/{{.ImagePullSecret}} -n {{.Namespace}} > /dev/null; then
  if [ -z "${REGISTRY_USERNAME}" ]; then
    echo -n "Username for {{.Registry}}: "
    read REGISTRY_USERNAME
    echo
  fi
  if [ -z "${REGISTRY_PASSWORD}" ]; then
    echo -n "Password for {{.Registry}}: "
    read -s REGISTRY_PASSWORD
    echo
  fi

  kubectl create secret docker-registry \
    "{{.ImagePullSecret}}" --namespace "{{.Namespace}}" \
    --docker-server={{.Registry}} \
    --docker-username="${REGISTRY_USERNAME}" \
    --docker-password="${REGISTRY_PASSWORD}" \
    --docker-email="support@stackrox.com"

	echo
fi

function print_rbac_instructions {
	echo
	echo "Error: Kubernetes RBAC configuration failed."
	echo "Specific errors are listed above."
	echo
	echo "You may need to elevate your privileges first:"
	echo "    kubectl create clusterrolebinding temporary-admin --clusterrole=cluster-admin --user you@example.com"
	echo
	echo "(Be sure to use the full username your cluster knows for you.)"
	echo
	echo "Then, rerun this script."
	echo
	echo "Finally, revoke your temporary privileges:"
	echo "    kubectl delete clusterrolebinding temporary-admin"
	echo
	echo "Contact your cluster administrator if you cannot obtain sufficient permission."
	exit 1
}

echo "Creating RBAC roles..."
kubectl apply -f "$DIR/sensor-rbac.yaml" || print_rbac_instructions

echo "Creating secrets for sensor..."
kubectl create secret -n "{{.Namespace}}" generic sensor-tls --from-file="$DIR/sensor-cert.pem" --from-file="$DIR/sensor-key.pem" --from-file="$DIR/central-ca.pem"

{{if .RuntimeSupport}}
echo "Creating secrets for collector..."
kubectl create secret -n "{{.Namespace}}" generic collector-tls --from-file="$DIR/collector-cert.pem" --from-file="$DIR/collector-key.pem" --from-file="$DIR/central-ca.pem"
{{- end}}

echo "Creating deployment..."
kubectl create -f "$DIR/sensor.yaml"
