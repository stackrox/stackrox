import { all, fork } from 'redux-saga/effects';

import alerts from './alertSagas';
import apiTokens from './apiTokenSagas';
import authProviders from './authSagas';
import clusters from './clusterSagas';
import deployments from './deploymentSagas';
import images from './imageSagas';
import policies from './policiesSagas';
import integrations from './integrationSagas';
import globalSearch from './globalSearchSagas';
import roles from './roleSagas';
import searches from './searchSagas';
import summaries from './summarySagas';
import secrets from './secretSagas';
import network from './networkSagas';
import metadata from './metadataSagas';
import processes from './processSagas';
import groups from './groupSagas';
import attributes from './attributesSagas';
import cli from './cliSagas';

export default function* root() {
    yield all([
        fork(alerts),
        fork(apiTokens),
        fork(authProviders),
        fork(cli),
        fork(clusters),
        fork(deployments),
        fork(images),
        fork(policies),
        fork(integrations),
        fork(globalSearch),
        fork(roles),
        fork(searches),
        fork(summaries),
        fork(secrets),
        fork(network),
        fork(metadata),
        fork(processes),
        fork(groups),
        fork(attributes)
    ]);
}
