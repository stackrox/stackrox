import { all, call, fork, put, select, takeLatest } from 'redux-saga/effects';
import * as service from 'services/GroupsService';
import { actions, types } from 'reducers/groups';
import { actions as authActions } from 'reducers/auth';
import { selectors } from 'reducers';
import { getGroupsWithDefault, getExistingGroupsWithDefault } from 'utils/permissionRuleGroupUtils';

import Raven from 'raven-js';

function* getRuleGroups() {
    try {
        const result = yield call(service.fetchGroups);
        yield put(actions.fetchGroups.success(result?.response || []));
    } catch (error) {
        yield put(actions.fetchGroups.failure(error));
    }
}

function* saveRuleGroup(action) {
    try {
        const { group, defaultRole, id } = action;
        const hasSomeInvalidRules = group.some(
            (rule) =>
                !rule?.props?.authProviderId ||
                !rule?.props?.key ||
                !rule?.props?.value ||
                !rule.roleName
        );
        if (!defaultRole || !id || hasSomeInvalidRules) {
            throw new Error(
                `Inconsistent state detected. Could not save auth provider minimum role and rules. Auth provider ID: ${id}, minimum role: ${defaultRole}, rules: ${JSON.stringify(
                    group
                )}`
            );
        }
        const existingGroups = yield select(selectors.getGroupsByAuthProviderId);
        const defaultGroup = yield call(service.getDefaultGroup, {
            authProviderId: id,
            roleName: defaultRole,
        });
        yield call(service.updateOrAddGroup, {
            requiredGroups: getGroupsWithDefault(group, id, defaultRole, defaultGroup?.response),
            previousGroups: getExistingGroupsWithDefault(existingGroups, id),
        });
        yield call(getRuleGroups);
        yield put(authActions.setAuthProviderEditingState(false));
        yield put(
            authActions.setSaveAuthProviderStatus({
                status: 'success',
                message: '',
            })
        );
    } catch (error) {
        yield put(
            authActions.setSaveAuthProviderStatus({
                status: 'error',
                message: error?.message,
            })
        );
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
        takeLatest(types.FETCH_RULE_GROUPS.REQUEST, getRuleGroups),
        fork(watchSaveRuleGroup),
        fork(watchDeleteRuleGroup),
    ]);
}
