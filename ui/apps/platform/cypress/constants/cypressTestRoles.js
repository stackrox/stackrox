import config from './cypressTestRoles.json';

export const TOKEN_NAME_PREFIX = config.tokenNamePrefix;
export const DEFAULT_ROLE = config.defaultRole;
export const cypressTestRoles = config.roles;

/**
 * @param {string} role An existing role name
 */
export function envVarKeyForRole(role) {
    return `ROX_AUTH_TOKEN_${role.replace(/\s+/g, '_').toUpperCase()}`;
}

/**
 * @param {string} role An existing role name
 */
export function tokenNameForRole(role) {
    return `${TOKEN_NAME_PREFIX}_${role}`;
}
