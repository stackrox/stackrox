
```
# Create core dump
$ kubectl exec sensor-676fd985fc-mxdv6 -- cd /stackrox/bin && /bin/bash -c "gcore 1"
# Copy core-dump
$ kubectl cp sensor-676fd985fc-mxdv6:/core.1 ./core.1 
```
