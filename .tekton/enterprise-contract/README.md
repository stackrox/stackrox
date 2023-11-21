# Enterprise Contract

This directory brings modified RHTAP Enterprise Contract (integration test) to allow CentOS Stream 9 as base image.
The non-hermetic EC configuration is saved as in `ec-no-hermetic-backup.yaml`.

## Installation instructions

Run the following against the RHTAP cluster:

```bash
oc apply -f acs-enterprise-contract-allow-quay-registry.yaml
```
