# Compliance Operator CRDs for Testing

The `crds` directory holds the [Compliance
Operator](https://github.com/ComplianceAsCode/compliance-operator) custom
resource definitions. They are applied by the sensor integration tests in
`sensor/tests/complianceoperator`, which need the CRDs to exist on the cluster
but do not need a running Compliance Operator.

This directory previously also contained a `create.sh` script and a `resources`
directory that fabricated scan results, making it look like the Compliance
Operator had scanned the cluster. Those were only used by the Compliance 1.0
integration tests and were removed along with Compliance 1.0.

The Compliance v2 tests do not use anything here. They install the Compliance
Operator (see `install_the_compliance_operator` in `tests/e2e/lib.sh`) and interact
with it directly.
