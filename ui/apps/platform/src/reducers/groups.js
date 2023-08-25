import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_RULE_GROUPS: createFetchingActionTypes('groups/FETCH_RULE_GROUPS'),
    SAVE_RULE_GROUP: 'groups/SAVE_RULE_GROUP',
    DELETE_RULE_GROUP: 'groups/DELETE_RULE_GROUP',
};

export const actions = {
    fetchGroups: createFetchingActions(types.FETCH_RULE_GROUPS),
    saveRuleGroup: (group, defaultRole, id) => ({
        type: types.SAVE_RULE_GROUP,
        group,
        defaultRole,
        id,
    }),
    deleteRuleGroup: (group) => ({
        type: types.DELETE_RULE_GROUP,
        group,
    }),
};

const groups = (state = [], action) => {
    if (action.type === types.FETCH_RULE_GROUPS.SUCCESS) {
        return action.response.groups;
    }
    return state;
};

const groupsByAuthProviderId = (state = {}, action) => {
    if (action.type === types.FETCH_RULE_GROUPS.SUCCESS) {
        const authProviderRuleGroups = {};
        action.response.groups.forEach((group) => {
            if (group && group.props) {
                if (!authProviderRuleGroups[group.props.authProviderId]) {
                    authProviderRuleGroups[group.props.authProviderId] = {
                        rules: [],
                        defaultRole: null,
                    };
                }
                if (group.props.key && group.props.key !== '') {
                    authProviderRuleGroups[group.props.authProviderId].rules.push(group);
                } else {
                    authProviderRuleGroups[group.props.authProviderId].defaultRole = group.roleName;
                    authProviderRuleGroups[group.props.authProviderId].defaultId = group.props.id;
                }
            }
        });
        return isEqual(authProviderRuleGroups, state) ? state : authProviderRuleGroups;
    }
    return state;
};

const reducer = combineReducers({
    groups,
    groupsByAuthProviderId,
});

const getRuleGroups = (state) => state.groups;

const getGroupsByAuthProviderId = (state) => state.groupsByAuthProviderId;

export const selectors = {
    getRuleGroups,
    getGroupsByAuthProviderId,
};

export default reducer;
