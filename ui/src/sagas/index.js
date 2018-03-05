import { all, fork } from 'redux-saga/effects';

import alerts from './alertSagas';
import benchmarks from './benchmarkSagas';
import clusters from './clusterSagas';

export default function* root() {
    yield all([fork(alerts), fork(benchmarks), fork(clusters)]);
}
