import queryString from 'qs';
import FileSaver from 'file-saver';

import { ListPolicy, Policy } from 'types/policy.proto';
import { addBrandedTimestampToString } from 'utils/dateUtils';
import { transformPolicyCriteriaValuesToStrings } from 'utils/policyUtils';

import axios from './instance';
import { Empty } from './types';

const baseUrl = '/v1/policies';

/*
 * Get a policy. Policy is a superset of ListPolicy.
 */
export function getPolicy(policyId: string): Promise<Policy> {
    return axios.get<Policy>(`${baseUrl}/${policyId}`).then((response) => response.data);
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
 * Reassess policies.
 */
export function reassessPolicies(): Promise<Empty> {
    return axios.post<Empty>(`${baseUrl}/reassess`).then((response) => response.data);
}

/*
 * Delete a policy.
 */
export function deletePolicy(policyId: string): Promise<Empty> {
    return axios.delete<Empty>(`${baseUrl}/${policyId}`).then((response) => response.data);
}

/*
 * Delete policies.
 */
export function deletePolicies(policyIds: string[] = []): Promise<Empty[]> {
    return Promise.all(policyIds.map((policyId) => deletePolicy(policyId)));
}

/*
 * Enable or disable notification to notifiers for a policy.
 */
export function enableDisablePolicyNotifications(
    policyId: string,
    notifierIds: string[],
    disable: boolean
): Promise<Empty> {
    return axios
        .patch<Empty>(`${baseUrl}/${policyId}/notifiers`, { notifierIds, disable })
        .then((response) => response.data);
}

/*
 * Enable or disable notification to notifiers for policies.
 */
export function enableDisableNotificationsForPolicies(
    policyIds: string[],
    notifierIds: string[],
    disable: boolean
): Promise<Empty[]> {
    return Promise.all(
        policyIds.map((policyId) =>
            enableDisablePolicyNotifications(policyId, notifierIds, disable)
        )
    );
}

/*
 * Save a policy.
 */
export function savePolicy(policy: Policy): Promise<Empty> {
    if (!policy.id) {
        throw new Error('Policy entity must have an id to be saved');
    }
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios
        .put<Empty>(`${baseUrl}/${policy.id}`, transformedPolicy)
        .then((response) => response.data);
}

/*
 * Create a new policy.
 */
export function createPolicy(policy: Policy): Promise<Policy> {
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios
        .post<Policy>(`${baseUrl}?enableStrictValidation=true`, transformedPolicy)
        .then((response) => response.data);
}

/*
 * Start a dry run for a policy. Return the jobId.
 */
export function startDryRun(policy: Policy): Promise<string> {
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios
        .post<{ jobId: string }>(`${baseUrl}/dryrunjob`, transformedPolicy)
        .then((response) => response.data.jobId);
}
/*
 * Get status of a dry run job.
 */
export function checkDryRun(jobId: string): Promise<DryRunJobStatusResponse> {
    return axios
        .get<DryRunJobStatusResponse>(`${baseUrl}/dryrunjob/${jobId}`)
        .then((response) => response.data);
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
export function cancelDryRun(jobId: string): Promise<Empty> {
    return axios.delete<Empty>(`${baseUrl}/dryrunjob/${jobId}`).then((response) => response.data);
}

/*
 * Update a policy to add deployment names into the exclusions.
 */
export async function excludeDeployments(
    policyId: string,
    deploymentNames: string[]
): Promise<Empty> {
    const policy = await getPolicy(policyId);

    const deploymentEntries = deploymentNames.map((name) => ({
        name: '',
        deployment: { name, scope: null },
        image: null,
        expiration: null,
    }));
    policy.exclusions = [...policy.exclusions, ...deploymentEntries];
    return axios.put<Empty>(`${baseUrl}/${policy.id}`, policy).then((response) => response.data);
}

/*
 * Enable or disable a policy.
 */
export function updatePolicyDisabledState(policyId: string, disabled: boolean): Promise<Empty> {
    return axios
        .patch<Empty>(`${baseUrl}/${policyId}`, { disabled })
        .then((response) => response.data);
}

/*
 * Enable or disable policies
 */
export function updatePoliciesDisabledState(
    policyIds: string[],
    disabled: boolean
): Promise<Empty[]> {
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
