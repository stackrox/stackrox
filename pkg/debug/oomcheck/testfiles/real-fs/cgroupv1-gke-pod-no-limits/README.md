This is `infra` pod from `infra-pr-777` GKE cluster (1.23.11-gke.300), it has empty `resources: {}` and the node has ~4 git of ram:

```bash
# From the node:
$ free -m
              total        used        free      shared  buff/cache   available
Mem:           3928         795        2239           3         893        2941
Swap:             0           0           0
```
