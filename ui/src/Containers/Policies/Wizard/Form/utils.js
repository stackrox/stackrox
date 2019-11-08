import policyFormFields from 'Containers/Policies/Wizard/Form/descriptors';
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

export function preFormatPolicyFields(policy) {
    let formattedPolicy = removeEmptyFields(policy);
    formattedPolicy = preFormatWhitelistField(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyFields(policy);
    serverPolicy = postFormatLifecycleField(serverPolicy);
    serverPolicy = postFormatEnforcementField(serverPolicy);
    serverPolicy = postFormatWhitelistField(serverPolicy);
    return serverPolicy;
}

export function mapDescriptorToKey(descriptor) {
    return descriptor.map(obj => obj.jsonpath);
}

export function getPolicyFormDataKeys() {
    const { policyDetails, policyConfiguration, policyStatus } = policyFormFields;
    const policyDetailsKeys = mapDescriptorToKey(policyDetails.descriptor);
    const policyConfigurationKeys = mapDescriptorToKey(policyConfiguration.descriptor);
    const policyStatusKeys = mapDescriptorToKey(policyStatus.descriptor);
    return [...policyDetailsKeys, ...policyConfigurationKeys, ...policyStatusKeys];
}
