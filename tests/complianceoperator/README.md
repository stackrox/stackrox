# Compliance V1 Integration Testing Data

The `create.sh` script simulates the installation of the [Compliance
Operator](https://github.com/ComplianceAsCode/compliance-operator) and creates
various resources that make it look like it scanned the cluster. These
resources are useful for testing against known state, making the tests more stable.

These resources are used in assertions in the
`tests/compliance_operator_tests.go` test cases, which are specific to the v1
integration of the Compliance Operator into stackrox/ACS.

The compliance v2 integration tests should not use these resources. Instead,
they install the Compliance Operator and perform integration tests by
interacting with the Compliance Operator directly.

This directory can be removed when Compliance v1 is no longer supported.
