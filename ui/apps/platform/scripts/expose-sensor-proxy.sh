#!/usr/bin/env bash

set -euo pipefail

# Default values
NAMESPACE="${1:-stackrox}"
HOURS="${2:-8}"

LB_SERVICE_NAME="sensor-proxy-dev-lb"
NETPOL_NAME="sensor-proxy-dev-allow-external"
CRONJOB_NAME="sensor-proxy-dev-cleanup"
SERVICE_NAME="sensor-proxy"

echo "Exposing sensor-proxy in namespace: $NAMESPACE"
echo "LoadBalancer will expire in: $HOURS hours"

# Validate namespace exists
if ! oc get namespace "$NAMESPACE" &>/dev/null; then
    echo "Error: Namespace '$NAMESPACE' does not exist."
    echo "Please create the namespace or specify a different namespace."
    exit 1
fi

# Validate sensor-proxy service exists
if ! oc -n "$NAMESPACE" get service "$SERVICE_NAME" &>/dev/null; then
    echo "Error: Service '$SERVICE_NAME' not found in namespace '$NAMESPACE'."
    echo "Please ensure StackRox secured cluster services are installed with the sensor-proxy enabled."
    exit 1
fi

# Calculate expiry time in ISO8601 format
EXPIRES_AT=$(date -u -v+"${HOURS}H" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+${HOURS} hours" +"%Y-%m-%dT%H:%M:%SZ")

echo "Expiry time: $EXPIRES_AT"

# Create or update NetworkPolicy to allow external access to port 9444
echo "Creating/updating NetworkPolicy '$NETPOL_NAME'..."

cat <<YAML | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: $NETPOL_NAME
  namespace: $NAMESPACE
  labels:
    stackrox.io/dev-route: "true"
    stackrox.io/managed-by: expose-sensor-proxy-script
  annotations:
    stackrox.io/expires-at: "$EXPIRES_AT"
    stackrox.io/created-by: expose-sensor-proxy.sh
spec:
  podSelector:
    matchLabels:
      app: sensor
  policyTypes:
  - Ingress
  ingress:
  - ports:
    - port: 9444
      protocol: TCP
YAML

# Create or update LoadBalancer Service
echo "Creating/updating LoadBalancer Service '$LB_SERVICE_NAME'..."

cat <<YAML | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  name: $LB_SERVICE_NAME
  namespace: $NAMESPACE
  labels:
    stackrox.io/dev-route: "true"
    stackrox.io/managed-by: expose-sensor-proxy-script
  annotations:
    stackrox.io/expires-at: "$EXPIRES_AT"
    stackrox.io/created-by: expose-sensor-proxy.sh
spec:
  type: LoadBalancer
  selector:
    app: sensor
  ports:
  - name: https
    port: 443
    targetPort: proxy-https
    protocol: TCP
YAML

# Create or update CronJob for cleanup
echo "Creating/updating CronJob '$CRONJOB_NAME'..."

cat <<YAML | oc apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: $CRONJOB_NAME
  namespace: $NAMESPACE
  labels:
    stackrox.io/dev-route: "true"
    stackrox.io/managed-by: expose-sensor-proxy-script
spec:
  schedule: "*/20 * * * *"
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 1
  concurrencyPolicy: Forbid
  jobTemplate:
    metadata:
      labels:
        stackrox.io/dev-route: "true"
        stackrox.io/managed-by: expose-sensor-proxy-script
    spec:
      template:
        metadata:
          labels:
            stackrox.io/dev-route: "true"
            stackrox.io/managed-by: expose-sensor-proxy-script
        spec:
          serviceAccountName: sensor
          restartPolicy: OnFailure
          containers:
          - name: cleanup
            image: quay.io/openshift/origin-cli:latest
            command:
            - /bin/bash
            - -c
            - |
              set -euo pipefail

              LB_SERVICE_NAME="$LB_SERVICE_NAME"
              NETPOL_NAME="$NETPOL_NAME"
              CRONJOB_NAME="$CRONJOB_NAME"
              NAMESPACE="$NAMESPACE"

              echo "Checking if LoadBalancer Service '\$LB_SERVICE_NAME' has expired..."

              # Get the expiry time from the Service annotation
              EXPIRES_AT=\$(oc get service "\$LB_SERVICE_NAME" -n "\$NAMESPACE" -o jsonpath='{.metadata.annotations.stackrox\.io/expires-at}' 2>/dev/null || echo "")

              if [ -z "\$EXPIRES_AT" ]; then
                echo "LoadBalancer Service '\$LB_SERVICE_NAME' not found or missing expiry annotation. Exiting."
                exit 0
              fi

              echo "LoadBalancer expires at: \$EXPIRES_AT"

              # Convert expiry time to epoch seconds
              EXPIRES_EPOCH=\$(date -d "\$EXPIRES_AT" +%s 2>/dev/null || echo "0")
              CURRENT_EPOCH=\$(date +%s)

              if [ "\$EXPIRES_EPOCH" -eq 0 ]; then
                echo "Error: Could not parse expiry time '\$EXPIRES_AT'"
                exit 1
              fi

              if [ "\$CURRENT_EPOCH" -ge "\$EXPIRES_EPOCH" ]; then
                echo "LoadBalancer has expired. Deleting resources..."
                oc delete service "\$LB_SERVICE_NAME" -n "\$NAMESPACE" --ignore-not-found=true
                oc delete networkpolicy "\$NETPOL_NAME" -n "\$NAMESPACE" --ignore-not-found=true
                oc delete cronjob "\$CRONJOB_NAME" -n "\$NAMESPACE" --ignore-not-found=true
                echo "Cleanup complete."
              else
                TIME_LEFT=\$((EXPIRES_EPOCH - CURRENT_EPOCH))
                HOURS_LEFT=\$((TIME_LEFT / 3600))
                MINUTES_LEFT=\$(((TIME_LEFT % 3600) / 60))
                echo "LoadBalancer has not expired yet. Time remaining: \${HOURS_LEFT}h \${MINUTES_LEFT}m"
              fi
YAML

# Wait for LoadBalancer to get an external IP
echo "Waiting for LoadBalancer to get an external IP..."
TIMEOUT=60
ELAPSED=0
EXTERNAL_IP=""

while [ $ELAPSED -lt $TIMEOUT ]; do
    EXTERNAL_IP=$(oc -n "$NAMESPACE" get service "$LB_SERVICE_NAME" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

    if [ -n "$EXTERNAL_IP" ]; then
        break
    fi

    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ -z "$EXTERNAL_IP" ]; then
    echo "Error: LoadBalancer did not receive an external IP within ${TIMEOUT} seconds."
    echo "Check service status: oc -n $NAMESPACE get service $LB_SERVICE_NAME"
    exit 1
fi

echo ""
echo "✓ Successfully created/updated LoadBalancer!"
echo ""
echo "The LoadBalancer and NetworkPolicy will expire at: $EXPIRES_AT"
echo "A CronJob will automatically clean up all resources when expired."
