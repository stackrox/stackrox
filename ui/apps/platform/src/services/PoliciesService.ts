import { normalize } from 'normalizr';
import queryString from 'qs';
import FileSaver from 'file-saver';

import { ListPolicy, Policy } from 'types/policy.proto';
import { addBrandedTimestampToString } from 'utils/dateUtils';
import { transformPolicyCriteriaValuesToStrings } from 'utils/policyUtils';

import axios from './instance';
import { policy as policySchema } from './schemas';

const baseUrl = '/v1/policies';
const policyCategoriesUrl = '/v1/policyCategories';

/*
 * Fetch policy summary for a given policy policyId.
 * TODO delete after policiesSagas.js has been deleted, because superseded by getPolicy
 */
export function fetchPolicy(
    policyId: string
): Promise<{ response: { entities: { policy: Record<string, Policy> } } }> {
    return axios.get<Policy>(`${baseUrl}/${policyId}`).then((response) => ({
        response: normalize(response.data, policySchema),
    }));
}

/*
 * Get a policy. Policy is a superset of ListPolicy.
 */
export function getPolicy(policyId: string): Promise<Policy> {
    return axios.get<Policy>(`${baseUrl}/${policyId}`).then((response) => response.data);
}

/*
 * Fetch a list of policies.
 * TODO delete after policiesSagas.js has been deleted, because superseded by getPolicies
 */
export function fetchPolicies(filters: { query: string }): Promise<{
    response: {
        entities: { policy: Record<string, ListPolicy> };
        result: { policies: ListPolicy[] };
    };
}> {
    const params = queryString.stringify(filters, { arrayFormat: 'repeat' });
    return axios.get<{ policies: Policy[] }>(`${baseUrl}?${params}`).then((response) => ({
        response: normalize(response.data, { policies: [policySchema] }),
    }));
}

/*
 * Get policies filtered by an optional query string. ListPolicy is a subset of Policy.
 */
export function getPolicies(query = ''): Promise<ListPolicy[]> {
    const params = queryString.stringify({ query });
    return axios
        .get<{ policies: Policy[] }>(`${baseUrl}?${params}`)
        .then((response) => response?.data?.policies ?? []);
}

/*
 * Fetch a list of policy categories.
 * TODO delete after policiesSagas.js has been deleted, because superseded by getPolicyCategories
 */
export function fetchPolicyCategories(): Promise<{ response: { categories: string[] } }> {
    return axios.get<{ categories: string[] }>(policyCategoriesUrl).then((response) => ({
        response: response.data,
    }));
}

/*
 * Get policy categories.
 */
export function getPolicyCategories(): Promise<string[]> {
    return axios
        .get<{ categories: string[] }>(policyCategoriesUrl)
        .then((response) => response?.data?.categories ?? []);
}

/*
 * Reassess policies.
 */
export function reassessPolicies(): Promise<Record<string, never>> {
    return axios
        .post<Record<string, never>>(`${baseUrl}/reassess`)
        .then((response) => response.data);
}

/*
 * Delete a policy.
 */
export function deletePolicy(policyId: string): Promise<Record<string, never>> {
    return axios
        .delete<Record<string, never>>(`${baseUrl}/${policyId}`)
        .then((response) => response.data);
}

/*
 * Delete policies.
 */
export function deletePolicies(policyIds: string[] = []): Promise<Record<string, never>[]> {
    return Promise.all(policyIds.map((policyId) => deletePolicy(policyId)));
}

/*
 * Enable or disable notification to notifiers for a policy.
 */
export function enableDisablePolicyNotifications(
    policyId: string,
    notifierIds: string[],
    disable: boolean
): Promise<Record<string, never>> {
    return axios
        .patch<Record<string, never>>(`${baseUrl}/${policyId}/notifiers`, { notifierIds, disable })
        .then((response) => response.data);
}

/*
 * Enable or disable notification to notifiers for policies.
 */
export function enableDisableNotificationsForPolicies(
    policyIds: string[],
    notifierIds: string[],
    disable: boolean
): Promise<Record<string, never>[]> {
    return Promise.all(
        policyIds.map((policyId) =>
            enableDisablePolicyNotifications(policyId, notifierIds, disable)
        )
    );
}

/*
 * Save a policy.
 */
export function savePolicy(policy: Policy): Promise<Record<string, never>> {
    if (!policy.id) {
        throw new Error('Policy entity must have an id to be saved');
    }
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios
        .put<Record<string, never>>(`${baseUrl}/${policy.id}`, transformedPolicy)
        .then((response) => response.data);
}

/*
 * Create a new policy.
 */
export function createPolicy(policy: Policy): Promise<{ data: Policy }> {
    /*
     * TODO after policiesSagas.js has been deleted:
     * function return type: Promise<Policy>
     * add method: then((response) => response.data)
     */
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios.post(`${baseUrl}?enableStrictValidation=true`, transformedPolicy); // TODO prop?
}

/*
 * Start a dry run for a policy. Return the jobId.
 */
export function startDryRun(policy: Policy): Promise<{ data: { jobId: string } }> {
    /*
     * TODO after policiesSagas.js has been deleted:
     * function return type: Promise<string>
     * add method: then((response) => response?.data?.jobId)
     */
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios.post(`${baseUrl}/dryrunjob`, transformedPolicy);
}

/*
 * Get status of a dry run job.
 */
export function checkDryRun(
    jobId: string
): Promise<{ data: { pending: boolean; result: { alerts: DryRunAlert[] } } }> {
    /*
     * TODO after policiesSagas.js has been deleted:
     * function return type: Promise<DryRunJobStatusResponse>
     * add method: then((response) => response.data)
     */
    return axios.get(`${baseUrl}/dryrunjob/${jobId}`);
}

export type DryRunJobStatusResponse = {
    pending: boolean;
    result: {
        alerts: DryRunAlert[];
    };
};

export type DryRunAlert = {
    deployment: string;
    violations: string[];
};

/*
 * Cancel a dry run job.
 */
export function cancelDryRun(jobId: string): Promise<Record<string, never>> {
    return axios
        .delete<Record<string, never>>(`${baseUrl}/dryrunjob/${jobId}`)
        .then((response) => response.data);
}

/*
 * Update a policy to add deployment names into the exclusions.
 */
export async function excludeDeployments(
    policyId: string,
    deploymentNames: string[]
): Promise<Record<string, never>> {
    const policy = await getPolicy(policyId);

    const deploymentEntries = deploymentNames.map((name) => ({
        name: '',
        deployment: { name, scope: null },
        image: null,
        expiration: null,
    }));
    policy.exclusions = [...policy.exclusions, ...deploymentEntries];
    return axios
        .put<Record<string, never>>(`${baseUrl}/${policy.id}`, policy)
        .then((response) => response.data);
}

/*
 * Enable or disable a policy.
 */
export function updatePolicyDisabledState(
    policyId: string,
    disabled: boolean
): Promise<Record<string, never>> {
    return axios
        .patch<Record<string, never>>(`${baseUrl}/${policyId}`, { disabled })
        .then((response) => response.data);
}

/*
 * Enable or disable policies
 */
export function updatePoliciesDisabledState(
    policyIds: string[],
    disabled: boolean
): Promise<Record<string, never>[]> {
    return Promise.all(policyIds.map((policyId) => updatePolicyDisabledState(policyId, disabled)));
}

/*
 * Export policies as JSON.
 */
export function exportPolicies(policyIds: string[]): Promise<void> {
    return axios.post(`${baseUrl}/export`, { policyIds }).then((response) => {
        if (response?.data?.policies?.length > 0) {
            try {
                const numSpaces = 4;
                const stringData = JSON.stringify(response.data, null, numSpaces);
                const filename = addBrandedTimestampToString('Exported_Policies');

                const file = new Blob([stringData], {
                    type: 'application/json',
                });

                FileSaver.saveAs(file, `${filename}.json`);
            } catch (error) {
                const message =
                    error instanceof Error
                        ? `Problem exporting policy data: ${error.message}`
                        : 'Problem exporting policy data';
                throw new Error(message);
            }
        } else {
            throw new Error('No policy data returned for the specified ID');
        }
    });
}

/*
 * Import policies.
 */
export function importPolicies(
    policies: Policy[],
    metadata: ImportPoliciesMetadata
): Promise<ImportPoliciesResponse> {
    return axios
        .post<ImportPoliciesResponse>(`${baseUrl}/import`, { policies, metadata })
        .then((response) => response?.data);
}

export type ImportPoliciesMetadata = {
    overwrite?: boolean;
};

export type ImportPoliciesResponse = {
    responses: ImportPolicyResponse[];
    allSucceeded: boolean;
};

export type ImportPolicyResponse = {
    succeeded: boolean;
    policy: Policy;
    errors: ImportPolicyError[];
};

export type ImportPolicyError =
    | {
          message: string;
          type: 'duplicate_id' | 'duplicate_name';
          duplicateName: string;
      }
    | {
          message: string;
          type: 'invalid_policy';
          validationError: string;
      };

/*
 * Create an unsaved policy object from a query string.
 */
export function generatePolicyFromSearch(searchParams: string): Promise<PolicyFromSearchResponse> {
    return axios
        .post<PolicyFromSearchResponse>(`${baseUrl}/from-search`, { searchParams })
        .then((response) => response?.data);
}

type PolicyFromSearchResponse = {
    policy: Policy;
    alteredSearchTerms: string[];
    hasNestedFields: boolean;
};
