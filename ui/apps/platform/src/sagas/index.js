import { all, fork } from 'redux-saga/effects';

import apiTokens from './apiTokenSagas';
import authProviders from './authSagas';
import clusterInitBundles from './clusterInitBundleSagas';
import integrations from './integrationSagas';
import roles from './roleSagas';
import searchAutoComplete from './searchAutocompleteSagas';
import metadata from './metadataSagas';
import groups from './groupSagas';

export default function* root() {
    yield all([
        fork(apiTokens),
        fork(authProviders),
        fork(clusterInitBundles),
        fork(integrations),
        fork(roles),
        fork(searchAutoComplete),
        fork(metadata),
        fork(groups),
    ]);
}
