package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// AuthMachineToMachine allows to exchange ID tokens for Central tokens without requiring user interaction.
var AuthMachineToMachine = registerFeature("Enable Auth Machine to Machine functionalities", "ROX_AUTH_MACHINE_TO_MACHINE", enabled)
