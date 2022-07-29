import axios from './instance';

const url = '/v1/groups';
const updateUrl = '/v1/groupsbatch';

/**
 * Fetches list of groups of rules
 *
 * @returns {Promise<Object, Error>} fulfilled with array of groups
 */
export function fetchGroups() {
    return axios.get(url).then((response) => ({
        response: response.data,
    }));
}

/**
 * Update/Add a group rule.
 *
 * @returns {Promise<Object, Error>}
 */
export function updateOrAddGroup({ oldGroups, newGroups }) {
    return axios.post(updateUrl, { previous_groups: oldGroups, required_groups: newGroups });
}

/**
 * Fetches the default rule.
 *
 * @returns {Promise<Object, Error>} fulfilled default group
 */
export function getDefaultGroup({ authProviderId, roleName }) {
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
export function deleteRuleGroup(data) {
    const { key, authProviderId, value, id } = data.props;
    return axios.delete(
        `${url}?authProviderId=${authProviderId}&key=${key}&value=${value}&id=${id}`,
        {}
    );
}
