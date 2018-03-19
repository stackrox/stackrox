const START = 'START';
const STOP = 'STOP';

/**
 * Polling action types.
 * @typedef {Object.<string, string>} PollingActionTypes
 * @property {string} START - start action type
 * @property {string} STOP - stop action type
 */

/**
 * Creates a map of action types with keys START and STOP and string values that use the given prefix.
 *
 * @param {string} prefix action names prefix
 * @returns {PollingActionTypes}
 */
export function createPollingActionTypes(prefix) {
    return {
        START: `${prefix}_${START}`,
        STOP: `${prefix}_${STOP}`
    };
}

function action(type, payload = {}) {
    return { type, ...payload };
}

/**
 * Polling actions.
 * @typedef {Object.<string, Function>} PollingActions
 * @property {Function} start - start action
 * @property {Function} stop - stop action
 */

/**
 * Creates a map of action functions for the given action types.
 *
 * @param {PollingActionTypes}
 * @returns {PollingActions}
 */
export function createPollingActions(types) {
    return {
        start: params => action(types.START, { params }),
        stop: () => action(types.STOP, {})
    };
}
