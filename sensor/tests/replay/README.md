# Sensor Replay Tests

## Record Events

The events located in `data` were recorded using the `local-sensor` tool.

### 1. Create the `policies.json` file containing the policies:

In order to reproduce the policies generated for the `replay test`:
1. Deploy ACS with the `deploy.sh` scripts (or any other way of your liking).
2. Activate all the default policies.
3. We added two new policies to test for specific violations regarding services and RBAC:
   1. `test-role`: 
      * Name: `test-role`
      * Severity: `Low`
      * Categories: `DevOps Best Practices`
      * Lifecycle stages: `Deploy`
      * Policy Section 1:
        * `RBAC permission level is at least`: `Elevated in the Namespace`
   2. `test-service`:
       * Name: `test-service`
       * Severity: `Low`
       * Categories: `DevOps Best Practices`
       * Lifecycle stages: `Deploy`
       * Policy Section 1:
           * `Exposed node port`: `30007`



Generate the `policies.json` file:
```
curl -k -u admin:$ROX_ADMIN_PASSWORD -H "Content-Type:application/json" https://$ROX_CENTRAL_ADDRESS/v1/policies > $tmpf
# and then
# for each id
curl -k -u admin:$ROX_ADMIN_PASSWORD -H "Content-Type:application/json" https://$ROX_CENTRAL_ADDRESS/v1/policies/$id >> $fname
```
### 2. Create the `trace.jsonl` file containing the k8s events:
```
go run tools/local-sensor/main.go -record -record-out=trace.jsonl -resync=0s
```

Resources used to trigger `test-service` and `test-role`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
spec:
  selector:
    matchLabels:
      app: nginx-1
      role: backend
  replicas: 1 
  template:
    metadata:
      labels:
        app: nginx-1
        role: backend
    spec:
      serviceAccountName: nginx-sa
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: svc-np
spec:
  type: NodePort
  selector:
    app: nginx-1
    role: backend
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
    nodePort: 30007
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: pod-privileged
rules:
- apiGroups: [""] 
  resources: ["pods"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: manage-pods
  namespace: default
subjects:
- kind: ServiceAccount
  name: nginx-sa
  namespace: default
roleRef:
  kind: Role 
  name: pod-privileged
  apiGroup: rbac.authorization.k8s.io
```

### 3. Create the `central-out.bin` file containing sensor's output events:
```
go run tools/local-sensor/main.go -replay -replay-in=trace.jsonl -resync=10s -delay=0s -format=raw -central-out=central-out.bin -with-policies=policies.json
```

## Manual Testing

Manual testing can be performed with:
```
go test -race -count=1 github.com/stackrox/rox/sensor/tests/replay
```