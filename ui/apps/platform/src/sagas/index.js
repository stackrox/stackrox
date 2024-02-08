import { all, fork } from 'redux-saga/effects';

import apiTokens from './apiTokenSagas';
import authProviders from './authSagas';
// import clusterInitBundles from './clusterInitBundleSagas';
import integrations from './integrationSagas';
import cloudSources from './cloudSourceSagas';
import roles from './roleSagas';
import searchAutoComplete from './searchAutocompleteSagas';
import metadata from './metadataSagas';
import groups from './groupSagas';

export default function* root() {
    yield all([
        fork(apiTokens),
        fork(authProviders),
        // Delete from reducers and sagas when we delete ROX_MOVE_INIT_BUNDLES_UI after 4.4 release.
        // fork(clusterInitBundles),
        fork(integrations),
        fork(cloudSources),
        fork(roles),
        fork(searchAutoComplete),
        fork(metadata),
        fork(groups),
    ]);
}
