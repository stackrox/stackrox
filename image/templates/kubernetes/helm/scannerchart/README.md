{{if eq .K8sConfig.DeploymentFormat.String "HELM"}}
# StackRox Scanner Chart

This Helm chart is for StackRox Scanner

Run the following command to render this chart:
- for Helm v2
```
helm install --name scanner .
```
- for Helm v3
```
helm install scanner .
```

{{- end}}
