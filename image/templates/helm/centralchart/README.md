{{if eq .K8sConfig.DeploymentFormat.String "HELM"}}
# StackRox Central Chart

This Helm chart is for StackRox Central

Run the following command to render this chart:
- for Helm v2
```
helm install --name central .
```
- for Helm v3
```
helm install central .
```

{{- end}}
