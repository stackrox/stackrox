import { all, call, fork, put, takeLatest, select } from 'redux-saga/effects';
import {
    createRole as serviceCreateRole,
    deleteRole as serviceDeleteRole,
    fetchRoles as serviceFetchRoles,
    updateRole as serviceUpdateRole,
} from 'services/RolesService';
import { actions, types } from 'reducers/roles';
import { selectors } from 'reducers';

import Raven from 'raven-js';
import { actions as notificationActions } from 'reducers/notifications';

function* getRoles() {
    try {
        const result = yield call(serviceFetchRoles);
        yield put(actions.fetchRoles.success(result?.response || []));
    } catch {
        // do nothing
    }
}

function* saveRole(action) {
    try {
        const { role } = action;
        const roles = yield select(selectors.getRoles);
        const isNewRole = !roles.filter((currRole) => currRole.name === role.name).length;
        if (isNewRole) {
            yield call(serviceCreateRole, role);
            yield put(actions.selectRole(role));
        } else {
            yield call(serviceUpdateRole, role);
            yield put(actions.selectRole(role));
        }
        yield call(getRoles);
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* deleteRole(action) {
    const { id } = action;
    try {
        yield call(serviceDeleteRole, id);
        yield put(actions.fetchRoles.request());
    } catch (error) {
        yield put(notificationActions.addNotification(error.response.data.error));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* selectRole(action) {
    const { role } = action;
    try {
        if (!role) {
            const roles = yield select(selectors.getRoles);
            yield put(actions.selectRole(roles[0]));
        }
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

function* watchSelectRole() {
    yield takeLatest(types.SELECTED_ROLE, selectRole);
}

export default function* integrations() {
    yield all([
        takeLatest(types.FETCH_ROLES.REQUEST, getRoles),
        fork(watchSaveRole),
        fork(watchDeleteRole),
        fork(watchSelectRole),
    ]);
}
