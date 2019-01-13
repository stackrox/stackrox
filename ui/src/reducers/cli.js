import { combineReducers } from 'redux';

// Action types

export const types = {
    TOGGLE_CLI_DOWNLOAD_VIEW: 'cli/TOGGLE_CLI_DOWNLOAD_VIEW',
    CLI_DOWNLOAD: 'cli/CLI_DOWNLOAD'
};

// Actions

export const actions = {
    toggleCLIDownloadView: () => ({
        type: types.TOGGLE_CLI_DOWNLOAD_VIEW
    }),
    downloadCLI: os => ({ type: types.CLI_DOWNLOAD, os })
};

// Reducers

const CLIDownloadView = (state = false, action) => {
    if (action.type === types.TOGGLE_CLI_DOWNLOAD_VIEW) {
        return !state;
    }
    return state;
};

const reducer = combineReducers({
    CLIDownloadView
});

// Selectors

const getCLIDownloadView = state => state.CLIDownloadView;

export const selectors = {
    getCLIDownloadView
};

export default reducer;
