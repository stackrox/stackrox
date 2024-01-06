This is taken from SSH session on a GKE (1.23.11-gke.300) node. As file contents show, there is no limit set for the session.

`free` output:

```bash
$ free -m
              total        used        free      shared  buff/cache   available
Mem:           3928         796        2237           3         894        2939
Swap:             0           0           0
```
