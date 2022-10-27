import { Group } from 'types/group.proto';

import axios from './instance';
import { Empty } from './types';

const url = '/v1/groups';
const updateUrl = '/v1/groupsbatch';

/**
 * Fetches list of groups of rules
 */
export function fetchGroups(): Promise<{ response: { groups: Group[] } }> {
    return axios.get<{ groups: Group[] }>(url).then((response) => ({
        response: response.data,
    }));
}

/**
 * Update all groups to add, delete, or update any group.
 */
export function updateGroups({ oldGroups, newGroups }: UpdateGroupsArg): Promise<Empty> {
    return axios
        .post<Empty>(updateUrl, { previous_groups: oldGroups, required_groups: newGroups })
        .then((response) => response.data);
}

type UpdateGroupsArg = {
    oldGroups: Group[];
    newGroups: Group[];
};

/**
 * Fetches the default rule.
 */
export function getDefaultGroup({
    authProviderId,
    roleName,
}: DefaultGroupArg): Promise<Group | undefined> {
    // The default group is characterized by the following:
    // - Only authProviderID is set, key and value are empty.
    // - The role name of the group matches the given role name.
    // We need _explicitly_ ask for empty key and value fields to receive the actual default role.
    return axios
        .get<{ groups: Group[] }>(
            `${url}?authProviderId=${authProviderId}&key=&value=&roleName=${roleName}`
        )
        .then((response) => response.data.groups[0]);
}

type DefaultGroupArg = {
    authProviderId: string;
    roleName: string;
};
