## Clair v4 Scanner Testing

The files in this directory deploy a Clair v4 scanner for test purposes. This is
intended to work with Clair v4.5.1.

To deploy the scanner, you need to run the deploy script. e.g.:

```
./scripts/ci/clairv4/deploy.sh qa-clairv4
```

To tear it down, run:

```
./scripts/ci/clairv4/teardown.sh qa-clairv4
```
