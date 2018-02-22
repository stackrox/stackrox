import { all, fork } from 'redux-saga/effects';

import alerts from './alertSagas';

export default function* root() {
    yield all([fork(alerts)]);
}
