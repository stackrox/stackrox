import {
    policyDetails,
    policyConfiguration,
    policyStatus
} from 'Containers/Policies/Wizard/Form/descriptors';
import removeEmptyFields from 'utils/removeEmptyFields';
import { clientOnlyWhitelistFieldNames } from './whitelistFieldNames';

function filterAndMapWhitelists(whitelists, filterFunc, mapFunc) {
    return whitelists && whitelists.length ? whitelists.filter(filterFunc).map(mapFunc) : [];
}

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    const clientPolicy = Object.assign({}, policy);

    clientPolicy[clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES] = filterAndMapWhitelists(
        whitelists,
        o => o.image && o.image.name,
        o => o.image.name
    );

    clientPolicy[
        clientOnlyWhitelistFieldNames.WHITELISTED_DEPLOYMENT_SCOPES
    ] = filterAndMapWhitelists(
        whitelists,
        o => o.deployment && (o.deployment.name || o.deployment.scope),
        o => o.deployment
    );

    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    serverPolicy.whitelists = [];

    const whitelistedDeploymentScopes =
        policy[clientOnlyWhitelistFieldNames.WHITELISTED_DEPLOYMENT_SCOPES];
    if (whitelistedDeploymentScopes && whitelistedDeploymentScopes.length) {
        serverPolicy.whitelists = serverPolicy.whitelists.concat(
            whitelistedDeploymentScopes.map(deployment => ({ deployment }))
        );
    }

    const whitelistedImageNames = policy[clientOnlyWhitelistFieldNames.WHITELISTED_IMAGE_NAMES];
    if (whitelistedImageNames && whitelistedImageNames.length > 0) {
        serverPolicy.whitelists = serverPolicy.whitelists.concat(
            whitelistedImageNames.map(name => ({ image: { name } }))
        );
    }

    return serverPolicy;
}

export function postFormatLifecycleField(policy) {
    const serverPolicy = Object.assign({}, policy);
    if (policy.lifecycleStages && policy.lifecycleStages.length !== 0)
        serverPolicy.lifecycleStages = policy.lifecycleStages.map(o => (o.value ? o.value : o));
    return serverPolicy;
}

export function postFormatEnforcementField(policy) {
    const serverPolicy = Object.assign({}, policy);
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

export function parseValueStr(value) {
    const valueArr = value.split('=');
    // for nested policy criteria fields
    if (valueArr.length === 2) {
        return {
            key: valueArr[0],
            value: valueArr[1]
        };
    }
    // for the Environment Variable policy criteria
    if (valueArr.length === 3) {
        return {
            source: valueArr[0],
            key: valueArr[1],
            value: valueArr[2]
        };
    }
    return {
        value
    };
}

function preFormatNestedPolicyFields(policy) {
    if (!policy.policy_sections) return policy;

    const clientPolicy = { ...policy };
    // itreating through each value in a policy group in a policy section to parse value string
    policy.policy_sections.forEach((policySection, sectionIdx) => {
        const { policy_groups: policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values } = policyGroup;
            values.forEach((value, valueIdx) => {
                clientPolicy.policy_sections[sectionIdx].policy_groups[groupIdx].values[
                    valueIdx
                ] = parseValueStr(value.value);
            });
        });
    });
    return clientPolicy;
}

export function formatValueStr({ source, key, value }) {
    let valueStr = value;
    if (source) {
        valueStr = `${source}=${key}=${value}`;
    } else if (key) {
        valueStr = `${key}=${value}`;
    }
    return valueStr;
}

function postFormatNestedPolicyFields(policy) {
    if (!policy.policy_sections) return policy;

    const serverPolicy = { ...policy };
    // itereating through each value in a policy group in a policy section to format to a flat value string
    policy.policy_sections.forEach((policySection, sectionIdx) => {
        const { policy_groups: policyGroups } = policySection;
        policyGroups.forEach((policyGroup, groupIdx) => {
            const { values } = policyGroup;
            values.forEach((value, valueIdx) => {
                serverPolicy.policy_sections[sectionIdx].policy_groups[groupIdx].values[
                    valueIdx
                ] = { value: formatValueStr(value) };
            });
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
    return descriptor.map(obj => obj.jsonpath);
}

export function getPolicyFormDataKeys() {
    const policyDetailsKeys = mapDescriptorToKey(policyDetails.descriptor);
    const policyConfigurationKeys = mapDescriptorToKey(policyConfiguration.descriptor);
    const policyStatusKeys = mapDescriptorToKey(policyStatus.descriptor);
    return [...policyDetailsKeys, ...policyConfigurationKeys, ...policyStatusKeys];
}
