## Test Data Sets

* 1.yaml

  This set was created using the following command:
  
  ```
  helm install --atomic --create-namespace -n stackrox --dry-run \
  -f deploy/common/local-dev-values.yaml --set imagePullSecrets.allowNone=true \
  stackrox-central-services stackrox-central-services-chart
  ```
