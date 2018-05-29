const REQUEST = 'REQUEST';
const SUCCESS = 'SUCCESS';
const FAILURE = 'FAILURE';

/**
 * Fetching action types.
 * @typedef {Object.<string, string>} FetchingActionTypes
 * @property {string} REQUEST - request action type
 * @property {string} SUCCESS - success action type
 * @property {string} FAILURE - failure action type
 */

/**
 * Creates a map of action types with keys REQUET, SUCCESS and FAILURE and string values that use the given prefix.
 *
 * @param {string} prefix action names prefix
 * @returns {FetchingActionTypes}
 */
export function createFetchingActionTypes(prefix) {
    return {
        REQUEST: `${prefix}_${REQUEST}`,
        SUCCESS: `${prefix}_${SUCCESS}`,
        FAILURE: `${prefix}_${FAILURE}`
    };
}

function action(type, payload = {}) {
    return { type, ...payload };
}

/**
 * Fetching actions.
 * @typedef {Object.<string, Function>} FetchingActions
 * @property {Function} request - request action (accepts params as a single parameter)
 * @property {Function} success - success action (accepts response as a single parameter)
 * @property {Function} failure - failure action (accepts error as a single parameter)
 */

/**
 * Creates a map of action functions for the given action types.
 *
 * @param {FetchingActionTypes}
 * @returns {FetchingActions}
 */
export function createFetchingActions(types) {
    return {
        request: params => action(types.REQUEST, { params }),
        success: (response, params) => action(types.SUCCESS, { response, params }),
        failure: (error, params) => action(types.FAILURE, { error, params })
    };
}

export function filterRequestActionTypes(type) {
    const matches = /(.*)_(REQUEST|SUCCESS|FAILURE)/.exec(type);
    if (!matches) return null;
    const [, requestName, requestState] = matches;
    return { requestName, requestState: requestState === 'REQUEST' };
}

export function getFetchingActionName(fetchingActionType) {
    const types = Object.keys(fetchingActionType);
    const matches = /(.*)_(REQUEST|SUCCESS|FAILURE)/.exec(fetchingActionType[types[0]]);
    if (!matches) return null;
    const [, actionName] = matches;
    return actionName;
}
