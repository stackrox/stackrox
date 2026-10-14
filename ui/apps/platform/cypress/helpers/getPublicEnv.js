const PUBLIC_ENV_KEYS = ['ORCHESTRATOR_FLAVOR'];

function isPublicEnvKey(key) {
    // Feature flag names come from /v1/featureflags at runtime, so match them by prefix.
    return PUBLIC_ENV_KEYS.includes(key) || (key.startsWith('ROX_') && key !== 'ROX_AUTH_TOKEN');
}

/**
 * Returns the subset of Cypress env that is safe to expose to the browser via Cypress.expose().
 * Cypress only fills expose from the config file or the --expose CLI flag, so cypress.config.js
 * uses this to copy the CYPRESS_* values exported by scripts/cypress.sh (feature flags,
 * ORCHESTRATOR_FLAVOR). Only allowlisted keys are copied, so secrets such as ROX_AUTH_TOKEN,
 * DEMO_PASSWORD, and the OCP console credentials stay available only through the async cy.env() command.
 */
function getPublicEnv(env = {}) {
    return Object.fromEntries(Object.entries(env).filter(([key]) => isPublicEnvKey(key)));
}

module.exports = { getPublicEnv };
