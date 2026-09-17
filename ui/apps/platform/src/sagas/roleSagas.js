import { all, call, put, takeLatest } from 'redux-saga/effects';
import { fetchRoles as serviceFetchRoles } from 'services/RolesService';
import { actions, types } from 'reducers/roles';

function* getRoles() {
    try {
        const result = yield call(serviceFetchRoles);
        yield put(actions.fetchRoles.success(result?.response || []));
    } catch {
        // do nothing
    }
}

export default function* integrations() {
    yield all([takeLatest(types.FETCH_ROLES.REQUEST, getRoles)]);
}
