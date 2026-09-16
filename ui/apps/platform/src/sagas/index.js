import { all, fork } from 'redux-saga/effects';

import authProviders from './authSagas';
import roles from './roleSagas';

import groups from './groupSagas';

export default function* root() {
    yield all([fork(authProviders), fork(roles), fork(groups)]);
}
