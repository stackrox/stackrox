import axios from './instance';

const resourcesUrl = '/v1/resources';

export function fetchResources(): Promise<string[]> {
    return axios
        .get<{ resources: string[] }>(resourcesUrl)
        .then((response) => response?.data?.resources ?? []);
}

const rolesUrl = '/v1/roles';

export type AccessLevel = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

/*
 * A permission associates a resource with its access level.
 */
export type PermissionsMap = Record<string, AccessLevel>;

export type Role = {
    name: string;
    // globalAccess is deprecated
    resourceToAccess: PermissionsMap; // deprecated: use only for classic UI
    description: string;
    permissionSetId: string;
    accessScopeId: string;
};

/**
 * Fetch entities and return object.response.roles :(
 */
export function fetchRoles(): Promise<{ response: { roles: Role[] } }> {
    return axios.get<{ roles: Role[] }>(rolesUrl).then((response) => ({
        response: response.data,
    }));
}

/*
 * Fetch entities and return array of objects.
 */
export function fetchRolesAsArray(): Promise<Role[]> {
    return axios.get<{ roles: Role[] }>(rolesUrl).then((response) => response?.data?.roles ?? []);
}

/*
 * Create entity and return empty object (unlike most create requests).
 */
export function createRole(entity: Role): Promise<Record<string, never>> {
    const { name } = entity;
    return axios.post(`${rolesUrl}/${name}`, entity);
}

/**
 * Update entity and return empty object.
 */
export function updateRole(entity: Role): Promise<Record<string, never>> {
    const { name } = entity;
    return axios.put(`${rolesUrl}/${name}`, entity);
}

/*
 * Delete entity which has name and return empty object.
 */
export function deleteRole(name: string): Promise<Record<string, never>> {
    return axios.delete(`${rolesUrl}/${name}`);
}

const permissionsURL = '/v1/mypermissions';

/**
 * Fetches current user's role permissions
 */
export function fetchUserRolePermissions(): Promise<{ response: Role }> {
    return axios.get<Role>(permissionsURL).then((response) => ({
        response: response.data,
    }));
}

const permissionSetsUrl = '/v1/permissionsets';

export type PermissionSet = {
    id: string;
    name: string;
    description: string;
    resourceToAccess: PermissionsMap;
};

/*
 * Fetch entities and return array of objects.
 */
export function fetchPermissionSets(): Promise<PermissionSet[]> {
    return axios
        .get<{ permissionSets: PermissionSet[] }>(permissionSetsUrl)
        .then((response) => response?.data?.permissionSets ?? []);
}

/*
 * Create entity and return object with id assigned by backend.
 */
export function createPermissionSet(entity: PermissionSet): Promise<PermissionSet> {
    return axios.post<PermissionSet>(permissionSetsUrl, entity).then((response) => response.data);
}

/*
 * Update entity and return empty object.
 */
export function updatePermissionSet(entity: PermissionSet): Promise<Record<string, never>> {
    const { id } = entity;
    return axios.put(`${permissionSetsUrl}/${id}`, entity);
}

/*
 * Delete entity which has id and return empty object.
 */
export function deletePermissionSet(id: string): Promise<Record<string, never>> {
    return axios.delete(`${permissionSetsUrl}/${id}`);
}
