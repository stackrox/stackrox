{{if eq .K8sConfig.DeploymentFormat.String "HELM"}}
# StackRox Monitoring Chart

This Helm chart is for StackRox Monitoring

Run the following command to render this chart:
- for Helm v2
```
helm install --name monitoring ./monitoring
```
- for Helm v3
```
helm install monitoring ./monitoring
```

{{- end}}