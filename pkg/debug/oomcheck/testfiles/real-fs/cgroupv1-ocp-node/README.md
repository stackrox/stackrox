This is taken from SSH session on OpenShift 4.11.18 node. There is no limit set for the session. The node has swap enabled but no swap device.

`free` output:

```bash
$ free -m
              total        used        free      shared  buff/cache   available
Mem:          14861        1933        1366          58       11560       12498
Swap:             0           0           0
```
