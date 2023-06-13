import axios from './instance';
import { Empty } from './types';
import { Traits } from '../types/traits.proto';

const accessScopessUrl = '/v1/simpleaccessscopes';

export const defaultAccessScopeIds = {
    Unrestricted: 'ffffffff-ffff-fff4-f5ff-ffffffffffff',
    DenyAll: 'ffffffff-ffff-fff4-f5ff-fffffffffffe',
};

// The only remaining usage of this function is in ResourceScopeSelection.tsx file,
// which will be deleted when Collections supersede Access Scopes in Vulnerability Reporting.
export function getIsDefaultAccessScopeId(id: string): boolean {
    return Object.values(defaultAccessScopeIds).includes(id);
}

export function getIsUnrestrictedAccessScopeId(id: string): boolean {
    return id === defaultAccessScopeIds.Unrestricted;
}

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

export type AccessScope = {
    id: string;
    name: string;
    description: string;
    rules: SimpleAccessScopeRules;
    traits?: Traits;
};

export const accessScopeNew: AccessScope = {
    id: '',
    name: '',
    description: '',
    rules: {
        includedClusters: [],
        includedNamespaces: [],
        clusterLabelSelectors: [],
        namespaceLabelSelectors: [],
    },
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
export function updateAccessScope(entity: AccessScope): Promise<Empty> {
    const { id } = entity;
    return axios.put(`${accessScopessUrl}/${id}`, entity);
}

/*
 * Delete entity which has id and return empty object.
 */
export function deleteAccessScope(id: string): Promise<Empty> {
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
        .then((response) => response?.data?.clusters ?? []);
}
