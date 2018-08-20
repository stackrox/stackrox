import { takeLatest, call, fork, put } from 'redux-saga/effects';

import fetchRoles from 'services/RolesService';
import { actions, types } from 'reducers/roles';

function* getRoles() {
    try {
        const result = yield call(fetchRoles);
        yield put(actions.fetchRoles.success(result.response));
    } catch (error) {
        yield put(actions.fetchRoles.failure(error));
    }
}

function* watchFetchRequest() {
    yield takeLatest(types.FETCH_ROLES.REQUEST, getRoles);
}

export default function* integrations() {
    yield fork(watchFetchRequest);
}
