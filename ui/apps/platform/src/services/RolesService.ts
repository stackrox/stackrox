import { Traits } from 'types/traits.proto';
import qs from 'qs';
import axios from './instance';
import { Empty } from './types';

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
    traits?: Traits;
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
export function createRole(entity: Role): Promise<Empty> {
    const { name } = entity;
    return axios.post(`${rolesUrl}/${name}`, entity);
}

/**
 * Update entity and return empty object.
 */
export function updateRole(entity: Role): Promise<Empty> {
    const { name } = entity;
    return axios.put(`${rolesUrl}/${name}`, entity);
}

/*
 * Delete entity which has name and return empty object.
 */
export function deleteRole(name: string): Promise<Empty> {
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
    traits?: Traits;
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
export function updatePermissionSet(entity: PermissionSet): Promise<Empty> {
    const { id } = entity;
    return axios.put(`${permissionSetsUrl}/${id}`, entity);
}

/*
 * Delete entity which has id and return empty object.
 */
export function deletePermissionSet(id: string): Promise<Empty> {
    return axios.delete(`${permissionSetsUrl}/${id}`);
}

const clustersForPermissionsUrl = '/v1/sac/clusters';

type ClustersForPermissionRequest = {
    permissions: string[];
};

// ScopeObject represents the (ID, name) pair identifying elements that belong to
// the access scope of a user.
type ScopeObject = {
    id: string;
    name: string;
};

// Aliases to increase readability of server responses with the same shape but different semantics.
export type ClusterScopeObject = ScopeObject;
export type NamespaceScopeObject = ScopeObject;

export type ClustersForPermissionsResponse = {
    clusters: ClusterScopeObject[];
};

export function getClustersForPermissions(
    permissions: string[]
): Promise<ClustersForPermissionsResponse> {
    const request: ClustersForPermissionRequest = { permissions };
    const params = qs.stringify(request, { arrayFormat: 'repeat' });
    return axios
        .get<ClustersForPermissionsResponse>(`${clustersForPermissionsUrl}?${params}`)
        .then((response) => response.data);
}

type NamespacesForClusterAndPermissionsRequest = {
    permissions: string[];
};

export type NamespacesForClusterAndPermissionsResponse = {
    namespaces: NamespaceScopeObject[];
};

export function getNamespacesForClusterAndPermissions(
    clusterID: string,
    permissions: string[]
): Promise<NamespacesForClusterAndPermissionsResponse> {
    const request: NamespacesForClusterAndPermissionsRequest = { permissions };
    const params = qs.stringify(request, { arrayFormat: 'repeat' });
    const targetUrl = `${clustersForPermissionsUrl}/${clusterID}/namespaces?${params}`;
    return axios
        .get<NamespacesForClusterAndPermissionsResponse>(targetUrl)
        .then((response) => response.data);
}
