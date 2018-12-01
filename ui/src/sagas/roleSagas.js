import { all, call, fork, put, takeLatest, select } from 'redux-saga/effects';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { accessControlPath } from 'routePaths';
import * as service from 'services/RolesService';
import { actions, types } from 'reducers/roles';
import { selectors } from 'reducers';

import Raven from 'raven-js';

function* getRoles() {
    try {
        const result = yield call(service.fetchRoles);
        yield put(actions.fetchRoles.success(result.response));
    } catch (error) {
        yield put(actions.fetchRoles.failure(error));
    }
}

function* saveRole(action) {
    try {
        const { role } = action;
        const roles = yield select(selectors.getRoles);
        const isNewRole = !roles.filter(currRole => currRole.name === role.name).length;
        if (isNewRole) {
            yield call(service.createRole, role);
            yield put(actions.selectRole(role));
        } else {
            yield call(service.updateRole, role);
            yield put(actions.selectRole(role));
        }
        yield call(getRoles);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* deleteRole(action) {
    const { id } = action;
    try {
        yield call(service.deleteRole, id);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* watchSaveRole() {
    yield takeLatest(types.SAVE_ROLE, saveRole);
}

function* watchDeleteRole() {
    yield takeLatest(types.DELETE_ROLE, deleteRole);
}

export default function* integrations() {
    yield all([
        takeEveryNewlyMatchedLocation(accessControlPath, getRoles),
        takeLatest(types.FETCH_ROLES.REQUEST, getRoles),
        fork(watchSaveRole),
        fork(watchDeleteRole)
    ]);
}
