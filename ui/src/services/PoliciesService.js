import { normalize } from 'normalizr';
import queryString from 'qs';
import FileSaver from 'file-saver';

import { addBrandedTimestampToString } from 'utils/dateUtils';
import { transformPolicyCriteriaValuesToStrings } from 'utils/policyUtils';
import axios from './instance';
import { policy as policySchema } from './schemas';

const baseUrl = '/v1/policies';
const policyCategoriesUrl = '/v1/policyCategories';

// this is the sample policy until BE is implemented and migrated to new policy format
// const samplePolicy = {
//     policy_version: '2.0',
//     id: '6a47dd95-6904-4997-ba21-c510ba3e75fa',
//     name: 'Registry Dependent CVSS',
//     description: 'Alert on vulnerabilities with varying CVSS scores depending on registry',
//     severity: 'HIGH_SEVERITY',
//     lifecycle_stages: ['BUILD', 'DEPLOY'],
//     categories: ['Security Best Practices'],
//     rationale: 'Demonstrate boolean policy logic',
//     remediation: 'Rebuild images',
//     policy_sections: [
//         {
//             section_name: 'Docker Registries',
//             policy_groups: [
// {
//     field_name: 'Dockerfile Line',
//     boolean_operator: 'OR',
//     negate: false,
//     values: [{ value: 'FROM=.*example.*' }]
// },
// {
//     field_name: 'Environment Variable',
//     boolean_operator: 'OR',
//     negate: false,
//     values: [{ value: 'SECRET_KEY=sampleKey=sampleValue' }]
// },
// {
//     field_name: 'CVSS',
//     boolean_operator: 'OR',
//     negate: false,
//     values: [{ value: '>=5' }]
// },
//         {
//             field_name: 'Image Registry',
//             boolean_operator: 'OR',
//             negate: false,
//             values: [{ value: 'docker.io' }],
//         },
//     ],
// },
//         {
//             section_name: 'Other Registries',
//             policy_groups: [
//                 {
//                     field_name: 'CVSS',
//                     boolean_operator: 'OR',
//                     negate: false,
//                     values: [{ value: '>=7' }]
//                 },
//                 {
//                     field_name: 'Image Registry',
//                     boolean_operator: 'OR',
//                     negate: false,
//                     values: [{ value: 'grc.io' }, { value: 'gke.grc.io' }, { value: 'quay.io' }]
//                 }
//             ]
//         },
//         {
//             section_name: 'StackRox Registries',
//             policy_groups: [
//                 {
//                     field_name: 'CVSS',
//                     boolean_operator: 'OR',
//                     negate: false,
//                     values: [{ value: '=10' }]
//                 },
//                 {
//                     field_name: 'Image Registry',
//                     boolean_operator: 'OR',
//                     negate: true,
//                     values: [{ value: 'stackrox.io' }, { value: 'collector.stackrox.io' }]
//                 }
//             ]
//         }
//     ],
// };

/**
 * Fetches policy summary for a given policy ID.
 * Returns normalized response with policy entity extracted.
 *
 * @param {!string} policyId
 * @returns {Promise<Object, Error>} fulfilled with normalized response
 */
export function fetchPolicy(policyId) {
    return axios.get(`${baseUrl}/${policyId}`).then((response) => ({
        response: normalize(response.data, policySchema),
    }));
}

/**
 * Fetches a list of policies.
 *
 * @param {!string} filters
 * @returns {Promise<Object, Error>} fulfilled with array of policies (as defined in .proto)
 */
export function fetchPolicies(filters) {
    const params = queryString.stringify({ ...filters }, { arrayFormat: 'repeat' });
    return axios.get(`${baseUrl}?${params}`).then((response) => ({
        response: normalize(response.data, { policies: [policySchema] }),
    }));
}

/**
 * Fetches a list of policy categories.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchPolicyCategories() {
    return axios.get(policyCategoriesUrl).then((response) => ({
        response: response.data,
    }));
}

/**
 * Reassesses policies.
 *
 * @returns {Promise<AxiosResponse, Error>}
 */
export function reassessPolicies() {
    return axios.post(`${baseUrl}/reassess`);
}

/**
 * Deletes a policy with a given id.
 *
 * @param {!string} policyId
 * @returns {Promise<AxiosResponse, Error>}
 */
export function deletePolicy(policyId) {
    return axios.delete(`${baseUrl}/${policyId}`);
}

/**
 * Deletes a list of policies by policyId.
 *
 * @param {string[]} policyIds
 * @returns {Promise<AxiosResponse, Error>}
 */
export function deletePolicies(policyIds = []) {
    return Promise.all(policyIds.map((policyId) => deletePolicy(policyId)));
}

/**
 * Enable / Disable notification to notifiers given by notifierIds for policy given by policyId.
 *
 * @param {!string} policyId
 * @param {!object} data
 * @returns {Promise<AxiosResponse, Error>}
 */
export function enableDisablePolicyNotifications(policyId, data) {
    return axios.patch(`${baseUrl}/${policyId}/notifiers`, data);
}

/**
 * Enable / Disable notification to notifiers given by notifierIds for list of policies given by policyIds.
 *
 * @param {!string[]} policyIds
 * @param {!string[]} notifierIds
 * @param {!boolean} disable
 * @returns {Promise<AxiosResponse, Error>}
 */
export function enableDisableNotificationsForPolicies(policyIds, notifierIds, disable) {
    const data = { notifierIds, disable };
    return Promise.all(
        policyIds.map((policyId) => enableDisablePolicyNotifications(policyId, data))
    );
}

/**
 * Saves a given policy.
 *
 * @param {!object} policy
 * @returns {Promise<AxiosResponse, Error>}
 */
export function savePolicy(policy) {
    if (!policy.id) throw new Error('Policy entity must have an id to be saved');
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios.put(`${baseUrl}/${policy.id}`, transformedPolicy);
}

/**
 * Creates a new policy.
 *
 * @param {!object} policy
 * @returns {Promise<AxiosResponse, Error>}
 */
export function createPolicy(policy) {
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios.post(`${baseUrl}`, transformedPolicy);
}

/**
 * Starts a dry run for a given policy.
 *
 * @param {!object} policy
 * @returns {Promise<AxiosResponse, Error>}
 */
export function startDryRun(policy) {
    const transformedPolicy = transformPolicyCriteriaValuesToStrings(policy);

    return axios.post(`${baseUrl}/dryrunjob`, transformedPolicy);
}

/**
 * Gets a dry run for a given job ID.
 *
 * @param {!string} jobId
 * @returns {Promise<AxiosResponse, Error>}
 */
export function checkDryRun(jobId) {
    return axios.get(`${baseUrl}/dryrunjob/${jobId}`);
}

/**
 * Cancels a dry run for a given job ID.
 *
 * @param {!string} jobId
 * @returns {Promise<AxiosResponse, Error>}
 */
export function cancelDryRun(jobId) {
    return axios.delete(`${baseUrl}/dryrunjob/${jobId}`);
}

/**
 * Updates policy with a given ID to add deployment into the whitelisted entries.
 *
 * @param {!string} policyId
 * @param {!string[]} deploymentNames
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export async function whitelistDeployments(policyId, deploymentNames) {
    const { response } = await fetchPolicy(policyId);
    const policy = response.entities.policy[policyId];

    const deploymentEntries = deploymentNames.map((name) => ({
        deployment: { name },
    }));
    policy.whitelists = [...policy.whitelists, ...deploymentEntries];
    return axios.put(`${baseUrl}/${policy.id}`, policy);
}

/**
 * Send request to enable / disable policy with a given ID.
 *
 * @param {!string} policyId
 * @param {!boolean} disabled if policy should be disabled
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function updatePolicyDisabledState(policyId, disabled) {
    return axios.patch(`${baseUrl}/${policyId}`, { disabled });
}

/**
 * Request policies as JSON for the given policy IDs.
 *
 * @param {!array} array of policyIds
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function exportPolicies(policyIds) {
    return axios.post(`${baseUrl}/export`, { policyIds }).then((response) => {
        if (response?.data && response?.data?.policies?.length > 0) {
            try {
                const numSpaces = 4;
                const stringData = JSON.stringify(response.data, null, numSpaces);
                const filename = addBrandedTimestampToString('Exported_Policies');

                const file = new Blob([stringData], {
                    type: 'application/json',
                });

                FileSaver.saveAs(file, `${filename}.json`);
            } catch (error) {
                throw new Error(`Problem saving policy data: ${error}`);
            }
        } else {
            throw new Error('No policy data returned for the specified ID');
        }
    });
}

/**
 * Request policies as JSON for the given policy IDs.
 *
 * @param {!array} array of policies
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function importPolicies(policies, metadata = {}) {
    return axios
        .post(`${baseUrl}/import`, { policies, metadata })
        .then((response) => response?.data);
}

/**
 * Create an unsaved policy object from a query string
 *
 * @param {!string}                         query string of search params
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function generatePolicyFromSearch(searchStr) {
    return axios
        .post(`${baseUrl}/from-search`, { searchParams: searchStr })
        .then((response) => response?.data);
}
