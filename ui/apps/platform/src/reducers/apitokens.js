import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const apiTokenFormId = 'api-token-form-id';

export const types = {
    FETCH_API_TOKENS: createFetchingActionTypes('apitokens/FETCH_API_TOKENS'),
    GENERATE_API_TOKEN: createFetchingActionTypes('apitokens/GENERATE_API_TOKEN'),
    REVOKE_API_TOKENS: 'apitokens/REVOKE_API_TOKENS',
    START_TOKEN_GENERATION_WIZARD: 'apitokens/START_TOKEN_GENERATION_WIZARD',
    CLOSE_TOKEN_GENERATION_WIZARD: 'apitokens/CLOSE_TOKEN_GENERATION_WIZARD',
};

export const actions = {
    fetchAPITokens: createFetchingActions(types.FETCH_API_TOKENS),
    generateAPIToken: createFetchingActions(types.GENERATE_API_TOKEN),
    revokeAPITokens: (ids) => ({ type: types.REVOKE_API_TOKENS, ids }),
    startTokenGenerationWizard: () => ({ type: types.START_TOKEN_GENERATION_WIZARD }),
    closeTokenGenerationWizard: () => ({ type: types.CLOSE_TOKEN_GENERATION_WIZARD }),
};

const apiTokens = (state = [], action) => {
    if (action.type === types.FETCH_API_TOKENS.SUCCESS) {
        return isEqual(action.response.tokens, state) ? state : action.response.tokens;
    }
    return state;
};

const tokenGenerationWizard = (state = null, { type, response }) => {
    switch (type) {
        case types.START_TOKEN_GENERATION_WIZARD:
            return { token: '', metadata: null };
        case types.CLOSE_TOKEN_GENERATION_WIZARD:
            return null;
        case types.GENERATE_API_TOKEN.SUCCESS:
            return { token: response.token, metadata: response.metadata };
        default:
            return state;
    }
};

const reducer = combineReducers({
    apiTokens,
    tokenGenerationWizard,
});

const getAPITokens = (state) => state.apiTokens;
const tokenGenerationWizardOpen = (state) => !!state.tokenGenerationWizard;
const getCurrentGeneratedToken = (state) =>
    state.tokenGenerationWizard ? state.tokenGenerationWizard.token : null;
const getCurrentGeneratedTokenMetadata = (state) =>
    state.tokenGenerationWizard ? state.tokenGenerationWizard.metadata : null;

export const selectors = {
    getAPITokens,
    tokenGenerationWizardOpen,
    getCurrentGeneratedToken,
    getCurrentGeneratedTokenMetadata,
};

export default reducer;
