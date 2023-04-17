/*
 * Return whether or not the testing environment has a feature flag.
 */
export function hasFeatureFlag(flag) {
    return Cypress.env(flag) || false;
}

export function hasOrchestratorFlavor(value) {
    return Cypress.env('ORCHESTRATOR_FLAVOR') === value;
}
