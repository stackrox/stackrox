import {
    policyDetails,
    policyConfiguration,
    policyStatus,
} from 'Containers/Policies/Wizard/Form/descriptors';
import removeEmptyFields from 'utils/removeEmptyFields';
import { keyBy } from 'lodash';
import { clientOnlyWhitelistFieldNames } from './whitelistFieldNames';

function filterAndMapWhitelists(whitelists, filterFunc, mapFunc) {
    return whitelists && whitelists.length ? whitelists.filter(filterFunc).map(mapFunc) : [];
}

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    const clientPolicy = { ...policy };

    clientPolicy[clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES] = filterAndMapWhitelists(
        whitelists,
        (o) => o.image && o.image.name,
        (o) => o.image.name
    );

    clientPolicy[
        clientOnlyWhitelistFieldNames.WHITELISTED_DEPLOYMENT_SCOPES
    ] = filterAndMapWhitelists(
        whitelists,
        (o) => o.deployment && (o.deployment.name || o.deployment.scope),
        (o) => o.deployment
    );

    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = { ...policy };
    serverPolicy.whitelists = [];

    const whitelistedDeploymentScopes =
        policy[clientOnlyWhitelistFieldNames.WHITELISTED_DEPLOYMENT_SCOPES];
    if (whitelistedDeploymentScopes && whitelistedDeploymentScopes.length) {
        serverPolicy.whitelists = serverPolicy.whitelists.concat(
            whitelistedDeploymentScopes.map((deployment) => ({ deployment }))
        );
    }

    const whitelistedImageNames = policy[clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES];
    if (whitelistedImageNames && whitelistedImageNames.length > 0) {
        serverPolicy.whitelists = serverPolicy.whitelists.concat(
            whitelistedImageNames.map((name) => ({ image: { name } }))
        );
    }

    return serverPolicy;
}

export function postFormatLifecycleField(policy) {
    const serverPolicy = { ...policy };
    if (policy.lifecycleStages && policy.lifecycleStages.length !== 0)
        serverPolicy.lifecycleStages = policy.lifecycleStages.map((o) => (o.value ? o.value : o));
    return serverPolicy;
}

export function postFormatEnforcementField(policy) {
    const serverPolicy = { ...policy };
    if (policy.enforcementActions) {
        if (typeof policy.enforcementActions === 'string') {
            serverPolicy.enforcementActions = [policy.enforcementActions];
        } else {
            // Already converted to array. No need to format.
            serverPolicy.enforcementActions = policy.enforcementActions;
        }
    }
    return serverPolicy;
}

// TODO: work with API to update contract for returning number comparison fields
//   until that improves, we short-circuit those fields here
const nonStandardNumberFields = [
    'CVSS',
    'Container CPU Request',
    'Container CPU Limit',
    'Container Memory Request',
    'Container Memory Limit',
];

const numericCompRe = /^([><=]+)?\D*(?=.)([+-]?([0-9]*)(\.([0-9]+))?)$/;

export function parseNumericComparisons(str) {
    const matches = str.match(numericCompRe);
    return [matches[1], matches[2]];
}
export function parseValueStr(value, fieldName) {
    if (typeof value !== 'string') return value;

    // TODO: work with API to update contract for returning number comparison fields
    //   until that improves, we short-circuit those fields here
    if (nonStandardNumberFields.includes(fieldName)) {
        const [comparison, num] = parseNumericComparisons(value);
        return comparison
            ? {
                  key: comparison,
                  value: num,
              }
            : {
                  key: '=',
                  value: num,
              };
    }
    // handle all other string fields
    const valueArr = value.split('=');
    // for nested policy criteria fields
    if (valueArr.length === 2) {
        return {
            key: valueArr[0],
            value: valueArr[1],
        };
    }
    // for the Environment Variable policy criteria
    if (valueArr.length === 3) {
        return {
            source: valueArr[0],
            key: valueArr[1],
            value: valueArr[2],
        };
    }
    return {
        value,
    };
}

function preFormatNestedPolicyFields(policy) {
    if (!policy.policySections) return policy;

    const clientPolicy = { ...policy };
    // itreating through each value in a policy group in a policy section to parse value string
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values } = policyGroup;
            values.forEach((value, valueIdx) => {
                clientPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[
                    valueIdx
                ] = parseValueStr(value.value, policyGroup.fieldName);
            });
        });
    });
    return clientPolicy;
}

export function formatValueStr({ source, key, value }, fieldName) {
    let valueStr = value;

    if (nonStandardNumberFields.includes(fieldName)) {
        // TODO: work with API to update contract for returning number comparison fields
        //   until that improves, we short-circuit those fields here
        valueStr = key !== '=' ? `${key} ${value}` : `${value}`;
    } else if (source) {
        valueStr = `${source}=${key}=${value}`;
    } else if (key) {
        valueStr = `${key}=${value}`;
    }
    return valueStr;
}

function postFormatNestedPolicyFields(policy) {
    if (!policy.policySections) return policy;

    const serverPolicy = { ...policy };
    // itereating through each value in a policy group in a policy section to format to a flat value string
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values } = policyGroup;
            values.forEach((value, valueIdx) => {
                serverPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[valueIdx] = {
                    value: formatValueStr(value, policyGroup.fieldName),
                };
            });
            delete serverPolicy.policySections[sectionIdx].policyGroups[groupIdx].fieldKey;
        });
    });
    return serverPolicy;
}

export function preFormatPolicyFields(policy) {
    let formattedPolicy = removeEmptyFields(policy);
    formattedPolicy = preFormatWhitelistField(formattedPolicy);
    formattedPolicy = preFormatNestedPolicyFields(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyFields(policy);
    serverPolicy = postFormatLifecycleField(serverPolicy);
    serverPolicy = postFormatEnforcementField(serverPolicy);
    serverPolicy = postFormatWhitelistField(serverPolicy);
    serverPolicy = postFormatNestedPolicyFields(serverPolicy);
    return serverPolicy;
}

export function mapDescriptorToKey(descriptor) {
    return descriptor.map((obj) => obj.jsonpath);
}

export function getPolicyFormDataKeys() {
    const policyDetailsKeys = mapDescriptorToKey(policyDetails.descriptor);
    const policyConfigurationKeys = mapDescriptorToKey(policyConfiguration.descriptor);
    const policyStatusKeys = mapDescriptorToKey(policyStatus.descriptor);
    return [...policyDetailsKeys, ...policyConfigurationKeys, ...policyStatusKeys];
}

export function getPolicyCriteriaFieldKeys(fields) {
    const fieldNameMap = keyBy(fields, (field) => field.field_name);
    const availableFieldKeys = [];
    policyConfiguration.descriptor.forEach((field) => {
        if (!fieldNameMap[field.name]) {
            availableFieldKeys.push(field.name);
        }
    });
    return availableFieldKeys;
}

export function addFieldArrayHandler(fields, value) {
    return () => fields.push(value);
}

export function removeFieldArrayHandler(fields, index) {
    return () => fields.remove(index);
}
