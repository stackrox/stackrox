import { all, fork } from 'redux-saga/effects';

import authProviders from './authSagas';
import roles from './roleSagas';
import searchAutoComplete from './searchAutocompleteSagas';

import groups from './groupSagas';

export default function* root() {
    yield all([fork(authProviders), fork(roles), fork(searchAutoComplete), fork(groups)]);
}
