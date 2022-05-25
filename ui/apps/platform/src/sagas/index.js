import { all, fork } from 'redux-saga/effects';

import apiTokens from './apiTokenSagas';
import authProviders from './authSagas';
import clusterInitBundles from './clusterInitBundleSagas';
import clusters from './clusterSagas';
import featureFlags from './featureFlagSagas';
import integrations from './integrationSagas';
import globalSearch from './globalSearchSagas';
import roles from './roleSagas';
import searches from './searchSagas';
import searchAutoComplete from './searchAutocompleteSagas';
import secrets from './secretSagas';
import network from './networkSagas';
import metadata from './metadataSagas';
import processes from './processSagas';
import groups from './groupSagas';
import attributes from './attributesSagas';
import systemConfig from './systemConfig';

export default function* root() {
    yield all([
        fork(apiTokens),
        fork(authProviders),
        fork(clusterInitBundles),
        fork(clusters),
        fork(featureFlags),
        fork(integrations),
        fork(globalSearch),
        fork(roles),
        fork(searches),
        fork(searchAutoComplete),
        fork(secrets),
        fork(network),
        fork(metadata),
        fork(processes),
        fork(groups),
        fork(attributes),
        fork(systemConfig),
    ]);
}
