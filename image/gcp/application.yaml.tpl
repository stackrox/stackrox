apiVersion: app.k8s.io/v1beta1
kind: Application
metadata:
  name: "$NAME"
  namespace: stackrox

spec:
  assemblyPhase: Pending
  selector:
    matchLabels:
      app.kubernetes.io/name: "$NAME"
