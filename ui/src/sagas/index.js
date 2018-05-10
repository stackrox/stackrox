import { all, fork } from 'redux-saga/effects';

import alerts from './alertSagas';
import authProviders from './authSagas';
import benchmarks from './benchmarkSagas';
import clusters from './clusterSagas';
import deployments from './deploymentSagas';
import images from './imageSagas';
import policies from './policiesSagas';
import integrations from './integrationSagas';
import globalSearch from './globalSearchSagas';
import searches from './searchSagas';
import summaries from './summarySagas';

export default function* root() {
    yield all([
        fork(alerts),
        fork(authProviders),
        fork(benchmarks),
        fork(clusters),
        fork(deployments),
        fork(images),
        fork(policies),
        fork(integrations),
        fork(globalSearch),
        fork(searches),
        fork(summaries)
    ]);
}
