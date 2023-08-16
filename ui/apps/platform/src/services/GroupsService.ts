import axios from './instance';
import { GetDefaultGroupRequest, Group, GroupBatchUpdateRequest } from '../types/group.proto';

const url = '/v1/groups';
const updateUrl = '/v1/groupsbatch';

/**
 * Fetches list of groups of rules
 *
 * @returns {Promise<Object, Error>} fulfilled with array of groups
 */
export function fetchGroups(): Promise<{ response: { groups: Group[] } }> {
    return axios.get(url).then((response) => ({
        response: response.data,
    }));
}

/**
 * Update/Add a group rule.
 *
 * @returns {Promise<Object, Error>}
 */
export function updateOrAddGroup(request: GroupBatchUpdateRequest) {
    return axios.post(updateUrl, request);
}

/**
 * Fetches the default rule.
 *
 * @returns {Promise<Object, Error>} fulfilled default group
 */
export function getDefaultGroup({ authProviderId, roleName }: GetDefaultGroupRequest) {
    // The default group is characterized by the following:
    // - Only authProviderID is set, key and value are empty.
    // - The role name of the group matches the given role name.
    // We need _explicitly_ ask for empty key and value fields to receive the actual default role.
    return axios
        .get(`${url}?authProviderId=${authProviderId}&key=&value=&roleName=${roleName}`)
        .then((response) => ({
            response: response.data?.groups[0],
        }));
}

/**
 * Deletes a group rule.
 *
 * @returns {Promise<Object, Error>}
 */
export function deleteRuleGroup(data: Group) {
    const { key, authProviderId, value, id } = data.props;
    return axios.delete(
        `${url}?authProviderId=${authProviderId}&key=${key}&value=${value}&id=${id}`,
        {}
    );
}
