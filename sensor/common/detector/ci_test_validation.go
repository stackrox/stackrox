package detector

// This file is used to validate CI test selection rules.
// It should trigger: sensor-integration-tests and go-postgres
// via the sensor-central-communication rule.
//
// Expected CI behavior:
// - go-postgres: RUN
// - sensor-integration-tests: RUN
// - Most E2E tests: RUN (default)
//
// This file can be safely deleted after CI validation.
