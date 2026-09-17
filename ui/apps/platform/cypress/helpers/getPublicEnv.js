const SENSITIVE_ENV_KEYS = [
    'ROX_AUTH_TOKEN',
    'OPENSHIFT_CONSOLE_USERNAME',
    'OPENSHIFT_CONSOLE_PASSWORD',
];

/**
 * Returns the subset of Cypress env that is safe to expose to the browser via Cypress.expose().
 * Cypress only fills expose from the config file or the --expose CLI flag, so cypress.config.js
 * uses this to copy the CYPRESS_* values exported by scripts/cypress.sh (feature flags,
 * ORCHESTRATOR_FLAVOR) while keeping secrets available only through the async cy.env() command.
 */
function getPublicEnv(env = {}) {
    return Object.fromEntries(
        Object.entries(env).filter(([key]) => !SENSITIVE_ENV_KEYS.includes(key))
    );
}

module.exports = { getPublicEnv, SENSITIVE_ENV_KEYS };
