import { DEFAULT_ROLE, cypressTestRoles, envVarKeyForRole } from '../constants/cypressTestRoles';

function validateRole(role) {
    if (!cypressTestRoles.includes(role)) {
        throw new Error(
            `withAuth/setAuth: unknown role "${role}". ` +
                `Available roles: ${cypressTestRoles.join(', ')}. ` +
                `Add the role to cypress/constants/cypressTestRoles.json if it is a new built-in role.`
        );
    }
}

function applyToken(role) {
    const envKey = envVarKeyForRole(role);
    const token = Cypress.env(envKey);
    if (token) {
        localStorage.setItem('access_token', token);
    } else {
        cy.log(`WARNING: ${envKey} is not set for role "${role}", tests will run unauthenticated`);
    }
}

/**
 * Sets up authentication for all tests in the current describe block via a beforeEach hook.
 * Must be called at the describe level, not inside an it() block.
 * @param {string} [role] - ACS role name from cypressTestRoles.json. Defaults to 'Admin'.
 */
export default function withAuth(role) {
    const effectiveRole = role ?? DEFAULT_ROLE;
    validateRole(effectiveRole);

    beforeEach(() => {
        applyToken(effectiveRole);
    });
}

/**
 * Switches the authenticated role mid-test. Call inside an it() block to change
 * which role is active for subsequent requests within that test.
 * @param {string} [role] - ACS role name from cypressTestRoles.json. Defaults to 'Admin'.
 */
export function setAuth(role) {
    const effectiveRole = role ?? DEFAULT_ROLE;
    validateRole(effectiveRole);
    applyToken(effectiveRole);
}
