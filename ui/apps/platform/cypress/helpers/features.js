/*
 * Return whether or not the testing environment has a feature flag.
 */
export function hasFeatureFlag(flag) {
    return Cypress.expose(flag) || false;
}

export function hasOrchestratorFlavor(value) {
    return Cypress.expose('ORCHESTRATOR_FLAVOR') === value;
}
