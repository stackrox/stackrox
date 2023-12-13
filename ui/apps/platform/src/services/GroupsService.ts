import { Group } from 'types/group.proto';
import axios from './instance';
import { Empty } from './types';

const url = '/v1/groups';
const updateUrl = '/v1/groupsbatch';

export type GetDefaultGroupRequest = {
    authProviderId: string;
    roleName: string;
};

export type GroupBatchUpdateRequest = {
    previousGroups: Group[];
    requiredGroups: Group[];
};

/**
 * Fetches list of groups of rules
 */
export function fetchGroups(): Promise<{ response: { groups: Group[] } }> {
    return axios.get(url).then((response) => ({
        response: response.data,
    }));
}

/**
 * Update/Add a group rule.
 */
export function updateOrAddGroup(request: GroupBatchUpdateRequest): Promise<Empty> {
    return axios.post<Empty>(updateUrl, request).then((response) => response.data);
}

/**
 * Fetches the default rule.
 */
export function getDefaultGroup({
    authProviderId,
    roleName,
}: GetDefaultGroupRequest): Promise<{ response: Group | undefined }> {
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
 */
export function deleteRuleGroup(data: Group) {
    const { key, authProviderId, value, id } = data.props;
    return axios.delete(
        `${url}?authProviderId=${authProviderId}&key=${key}&value=${value}&id=${id}`,
        {}
    );
}
