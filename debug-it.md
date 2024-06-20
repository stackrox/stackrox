
```
# get sensor pod name
$ sensor_pod_name =$(kubectl -n stackrox get pods -l app=sensor custom-columns=NAME:.metadata.name)

# Create core dump
$ kubectl exec $(sensor_pod_name) -- cd /var/stackrox/log && /bin/bash -c "gcore 1"

# Copy core-dump
$ kubectl cp $(sensor_pod_name):/var/stackrox/log/core.1 ./core.1 
```
