import { all, fork } from 'redux-saga/effects';

import alerts from './alertSagas';
import authProviders from './authSagas';
import benchmarks from './benchmarkSagas';
import clusters from './clusterSagas';
import deployments from './riskSagas';
import integrations from './integrationSagas';

export default function* root() {
    yield all([
        fork(alerts),
        fork(authProviders),
        fork(benchmarks),
        fork(clusters),
        fork(deployments),
        fork(integrations)
    ]);
}
