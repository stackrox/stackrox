# Kubernetes APIs for Central

## Quick start

```
To generate APIs run:
$ sed '1s/github.com\/stackrox\/rox/github.com\/stackrox\/stackrox/g' go.mod > go.mod.tmp
$ mv go.mod go.mod.bk
$ mv go.mod.tmp go.mod
$ ./apis/hack/update-codegen.sh
$ mv go.mod.bk go.mod
```

## Generate APIs - How to

**Resources:**

- Blog Article: https://cloud.redhat.com/blog/kubernetes-deep-dive-code-generation-customresources
- Example: https://github.com/kubernetes/sample-controller
- Repo: https://github.com/kubernetes/code-generator
