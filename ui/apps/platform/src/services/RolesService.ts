import axios from './instance';

const resourcesUrl = '/v1/resources';

export function fetchResources(): Promise<{ response: { resources: string[] } }> {
    return axios.get<{ resources: string[] }>(resourcesUrl).then((response) => ({
        response: response.data,
    }));
}

// TODO After classic Access Control code has been deleted,
// delete preceding function fetchResources also its Redux support,
// and then rename the following function as fetchResources.
export function fetchResourcesAsArray(): Promise<string[]> {
    return axios
        .get<{ resources: string[] }>(resourcesUrl)
        .then((response) => response.data.resources);
}

const rolesUrl = '/v1/roles';

export type AccessLevel = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

export type Role = {
    name: string;
    // globalAccess is deprecated
    resourceToAccess: Record<string, AccessLevel>; // deprecated: use only for classic UI
    id: string;
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
    return axios.get<{ roles: Role[] }>(rolesUrl).then((response) => response.data.roles);
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
    minimumAccessLevel: AccessLevel;
    resourceToAccess: Record<string, AccessLevel>;
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

const accessScopessUrl = '/v1/simpleaccessscopes';

export type SimpleAccessScopeNamespace = {
    clusterName: string;
    namespaceName: string;
};

/*
 * For more information about label selectors:
 * https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
 */

export type LabelSelectorOperator = 'UNKNOWN' | 'IN' | 'NOT_IN' | 'EXISTS' | 'NOT_EXISTS';

export type LabelSelectorRequirement = {
    key: string;
    op: LabelSelectorOperator;
    values: string[];
};

export type LabelSelector = {
    requirements: LabelSelectorRequirement[];
};

export type LabelSelectorsKey = 'clusterLabelSelectors' | 'namespaceLabelSelectors';

export type SimpleAccessScopeRules = {
    includedClusters: string[];
    includedNamespaces: SimpleAccessScopeNamespace[];
    clusterLabelSelectors: LabelSelector[];
    namespaceLabelSelectors: LabelSelector[];
};

export function getIsKeyExistsOperator(op: LabelSelectorOperator): boolean {
    return op === 'EXISTS' || op === 'NOT_EXISTS';
}

export function getIsKeyInSetOperator(op: LabelSelectorOperator): boolean {
    return op === 'IN' || op === 'NOT_IN';
}

/*
 * A valid "key in set" requirement has at least one value.
 */
export function getIsValidRequirement({ op, values }: LabelSelectorRequirement): boolean {
    return !getIsKeyInSetOperator(op) || values.length !== 0;
}

/*
 * A valid label selector has at least one requirement.
 */
export function getIsValidRequirements(requirements: LabelSelectorRequirement[]): boolean {
    return requirements.length !== 0 && requirements.every(getIsValidRequirement);
}

export function getIsValidLabelSelectors(labelSelectors: LabelSelector[]): boolean {
    return labelSelectors.every(({ requirements }) => getIsValidRequirements(requirements));
}

export function getIsValidRules({
    clusterLabelSelectors,
    namespaceLabelSelectors,
}: SimpleAccessScopeRules): boolean {
    return (
        getIsValidLabelSelectors(clusterLabelSelectors) &&
        getIsValidLabelSelectors(namespaceLabelSelectors)
    );
}

export type AccessScope = {
    id: string;
    name: string;
    description: string;
    rules: SimpleAccessScopeRules;
};

/*
 * Fetch entities and return array of objects.
 */
export function fetchAccessScopes(): Promise<AccessScope[]> {
    return axios
        .get<{ accessScopes: AccessScope[] }>(accessScopessUrl)
        .then((response) => response?.data?.accessScopes ?? []);
}

/*
 * Create entity and return object with id assigned by backend.
 */
export function createAccessScope(entity: AccessScope): Promise<AccessScope> {
    return axios.post<AccessScope>(accessScopessUrl, entity).then((response) => response.data);
}

/*
 * Update entity and return empty object.
 */
export function updateAccessScope(entity: AccessScope): Promise<Record<string, never>> {
    const { id } = entity;
    return axios.put(`${accessScopessUrl}/${id}`, entity);
}

/*
 * Delete entity which has id and return empty object.
 */
export function deleteAccessScope(id: string): Promise<Record<string, never>> {
    return axios.delete(`${accessScopessUrl}/${id}`);
}

const computeEffectiveAccessScopeUrl = '/v1/computeeffectiveaccessscope';

export type EffectiveAccessScopeDetail = 'STANDARD' | 'MINIMAL' | 'HIGH';

export type EffectiveAccessScopeState = 'UNKNOWN' | 'INCLUDED' | 'EXCLUDED' | 'PARTIAL';

export type EffectiveAccessScopeNamespace = {
    id: string;
    name: string;
    state: EffectiveAccessScopeState;
    labels: Record<string, string>;
};

export type EffectiveAccessScopeCluster = {
    id: string;
    name: string;
    state: EffectiveAccessScopeState;
    namespaces: EffectiveAccessScopeNamespace[];
    labels: Record<string, string>;
};

export type EffectiveAccessScope = {
    clusters: EffectiveAccessScopeCluster[];
};

/*
 * Given rules from simple access scope and detail option,
 * return effective access scope for clusters (which include namespaces).
 */
export function computeEffectiveAccessScopeClusters(
    simpleRules: SimpleAccessScopeRules,
    detail: EffectiveAccessScopeDetail = 'HIGH'
): Promise<EffectiveAccessScopeCluster[]> {
    return axios
        .post<EffectiveAccessScope>(
            computeEffectiveAccessScopeUrl,
            {
                simpleRules,
            },
            {
                params: {
                    detail,
                },
            }
        )
        .then((response) => response.data.clusters);
}
