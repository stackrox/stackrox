import axios from './instance';

const rolesUrl = '/v1/roles';
const permissionsURL = '/v1/mypermissions';
const resourcesURL = '/v1/resources';

export type Access = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

export type Role = {
    name: string;
    globalAccess: Access;
    resourceToAccess: Record<string, Access>;
};

export function fetchResources(): Promise<{ response: { resources: string[] } }> {
    return axios.get<{ resources: string[] }>(resourcesURL).then((response) => ({
        response: response.data,
    }));
}

/**
 * Fetches list of roles
 */
export function fetchRoles(): Promise<{ response: { roles: Role[] } }> {
    return axios.get<{ roles: Role[] }>(rolesUrl).then((response) => ({
        response: response.data,
    }));
}

/**
 * Fetches current user's role permissions
 */
export function fetchUserRolePermissions(): Promise<{ response: Role }> {
    return axios.get<Role>(permissionsURL).then((response) => ({
        response: response.data,
    }));
}

/**
 * Creates a role.
 */
export function createRole(data: Role): Promise<Role> {
    const { name } = data;
    return axios.post(`${rolesUrl}/${name}`, data);
}

/**
 * Updates a role.
 */
export function updateRole(data: Role): Promise<Role> {
    const { name } = data;
    return axios.put(`${rolesUrl}/${name}`, data);
}

/**
 * Deletes a role. Returns an empty object.
 */
export function deleteRole(name: string): Promise<Record<string, never>> {
    return axios.delete(`${rolesUrl}/${name}`);
}
