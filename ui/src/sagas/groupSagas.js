import { all, call, fork, put, select, takeLatest } from 'redux-saga/effects';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { accessControlPath } from 'routePaths';
import * as service from 'services/GroupsService';
import { actions, types } from 'reducers/groups';
import { selectors } from 'reducers';
import { getGroupsWithDefault, getExistingGroupsWithDefault } from 'utils/permissionRuleGroupUtils';

import Raven from 'raven-js';

function* getRuleGroups() {
    try {
        const result = yield call(service.fetchGroups);
        yield put(actions.fetchGroups.success(result.response));
    } catch (error) {
        yield put(actions.fetchGroups.failure(error));
    }
}

function* saveRuleGroup(action) {
    try {
        const { group, defaultRole, id } = action;
        const selectedAuthProvider = yield select(selectors.getSelectedAuthProvider);
        const existingGroups = yield select(selectors.getGroupsByAuthProviderId);
        yield call(service.updateOrAddGroup, {
            newGroups: getGroupsWithDefault(group, selectedAuthProvider.id || id, defaultRole),
            oldGroups: getExistingGroupsWithDefault(existingGroups, selectedAuthProvider.id || id)
        });
        yield call(getRuleGroups);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* deleteRuleGroup(action) {
    const { group } = action;
    try {
        yield call(service.deleteRuleGroup, group);
        yield call(getRuleGroups);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* watchSaveRuleGroup() {
    yield takeLatest(types.SAVE_RULE_GROUP, saveRuleGroup);
}

function* watchDeleteRuleGroup() {
    yield takeLatest(types.DELETE_RULE_GROUP, deleteRuleGroup);
}

export default function* groups() {
    yield all([
        takeEveryNewlyMatchedLocation(accessControlPath, getRuleGroups),
        takeLatest(types.FETCH_RULE_GROUPS.REQUEST, getRuleGroups),
        fork(watchSaveRuleGroup),
        fork(watchDeleteRuleGroup)
    ]);
}
