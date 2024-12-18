import { all, fork } from 'redux-saga/effects';

import apiTokens from './apiTokenSagas';
import authProviders from './authSagas';
import machineAccessConfigs from './machineAccessSagas';
import integrations from './integrationSagas';
import cloudSources from './cloudSourceSagas';
import roles from './roleSagas';
import searchAutoComplete from './searchAutocompleteSagas';
import metadata from './metadataSagas';
import groups from './groupSagas';

export default function* root(history) {
    yield all([
        fork(apiTokens),
        fork(authProviders, history),
        fork(machineAccessConfigs),
        fork(integrations),
        fork(cloudSources),
        fork(roles),
        fork(searchAutoComplete),
        fork(metadata),
        fork(groups),
    ]);
}
