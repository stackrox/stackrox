{{if eq .K8sConfig.DeploymentFormat.String "HELM"}}
# StackRox Monitoring Chart

This Helm chart is for StackRox Monitoring

You can render this chart with
```
helm install --name monitoring .
```
{{- end}}