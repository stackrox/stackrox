import keyBy from 'lodash/keyBy';

import { removeEmptyPolicyFields } from 'utils/policyUtils';
import { clientOnlyExclusionFieldNames } from './whitelistFieldNames';
import {postFormatImageSigningPolicyGroup} from "../../PatternFly/policies.utils";

function filterAndMapExclusions(exclusions, filterFunc, mapFunc) {
    return exclusions && exclusions.length ? exclusions.filter(filterFunc).map(mapFunc) : [];
}

export function preFormatExclusionField(policy) {
    const { exclusions } = policy;
    const clientPolicy = { ...policy };

    clientPolicy[clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES] = filterAndMapExclusions(
        exclusions,
        (o) => o.image && o.image.name,
        (o) => o.image.name
    );

    clientPolicy[clientOnlyExclusionFieldNames.EXCLUDED_DEPLOYMENT_SCOPES] = filterAndMapExclusions(
        exclusions,
        (o) => o.deployment && (o.deployment.name || o.deployment.scope),
        (o) => o.deployment
    );

    return clientPolicy;
}

export function postFormatExclusionField(policy) {
    const serverPolicy = { ...policy };
    serverPolicy.exclusions = [];

    const excludedDeploymentScopes =
        policy[clientOnlyExclusionFieldNames.EXCLUDED_DEPLOYMENT_SCOPES];
    if (excludedDeploymentScopes && excludedDeploymentScopes.length) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedDeploymentScopes.map((deployment) => ({ deployment }))
        );
    }

    const excludedImageNames = policy[clientOnlyExclusionFieldNames.EXCLUDED_IMAGE_NAMES];
    if (excludedImageNames && excludedImageNames.length > 0) {
        serverPolicy.exclusions = serverPolicy.exclusions.concat(
            excludedImageNames.map((name) => ({ image: { name } }))
        );
    }

    return serverPolicy;
}

export function postFormatLifecycleField(policy) {
    const serverPolicy = { ...policy };
    if (policy.lifecycleStages && policy.lifecycleStages.length !== 0) {
        serverPolicy.lifecycleStages = policy.lifecycleStages.map((o) => (o.value ? o.value : o));
    }
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
    'Replicas',
    'Severity',
];

function isCompoundField(fieldName = '') {
    const compoundValueFields = [
        'Disallowed Annotation',
        'Disallowed Image Label',
        'Dockerfile Line',
        'Environment Variable',
        'Image Component',
        'Required Annotation',
        'Required Image Label',
        'Required Label',
    ];

    return compoundValueFields.includes(fieldName);
}

const numericCompRe =
    /^([><=]+)?\D*(?=.)(([+-]?([0-9]*)(\.([0-9]+))?)|(UNKNOWN|LOW|MODERATE|IMPORTANT|CRITICAL))$/;

export function parseNumericComparisons(str) {
    const matches = str.match(numericCompRe);
    return [matches[1], matches[2]];
}
export function parseValueStr(value, fieldName) {
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
    if (typeof value === 'string' && isCompoundField(fieldName)) {
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
    }
    return {
        value,
    };
}

function preFormatNestedPolicyFields(policy) {
    if (!policy.policySections) {
        return policy;
    }

    const clientPolicy = { ...policy };
    // itreating through each value in a policy group in a policy section to parse value string
    policy.policySections.forEach((policySection, sectionIdx) => {
        const { policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values, fieldName } = policyGroup;
            values.forEach((value, valueIdx) => {
                clientPolicy.policySections[sectionIdx].policyGroups[groupIdx].values[valueIdx] =
                    parseValueStr(value.value, fieldName);
            });
        });
    });
    return clientPolicy;
}

export function formatValueStr(valueObj, fieldName) {
    if (!valueObj) {
        return valueObj;
    }
    const { source, key, value } = valueObj;
    let valueStr = value;

    if (nonStandardNumberFields.includes(fieldName)) {
        // TODO: work with API to update contract for returning number comparison fields
        //   until that improves, we short-circuit those fields here
        valueStr = key !== '=' ? `${key} ${value}` : `${value}`;
    } else if (source || fieldName === 'Environment Variable') {
        valueStr = `${source || ''}=${key}=${value}`;
    } else if (key) {
        valueStr = `${key}=${value}`;
    }
    return valueStr;
}

function postFormatNestedPolicyFields(policy) {
    if (!policy.policySections) {
        return policy;
    }

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
    let formattedPolicy = removeEmptyPolicyFields(policy);
    formattedPolicy = preFormatExclusionField(formattedPolicy);
    formattedPolicy = preFormatNestedPolicyFields(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyPolicyFields(policy);
    serverPolicy = postFormatLifecycleField(serverPolicy);
    serverPolicy = postFormatEnforcementField(serverPolicy);
    serverPolicy = postFormatExclusionField(serverPolicy);
    serverPolicy = postFormatImageSigningPolicyGroup(serverPolicy);
    serverPolicy = postFormatNestedPolicyFields(serverPolicy);
    return serverPolicy;
}

export function getPolicyCriteriaFieldKeys(fields, descriptor) {
    const fieldNameMap = keyBy(fields, (field) => field.field_name);
    const availableFieldKeys = [];
    descriptor.forEach((field) => {
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
