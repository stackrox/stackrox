import policyFormFields from 'Containers/Policies/Wizard/Form/descriptors';
import removeEmptyFields from 'utils/removeEmptyFields';

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    const clientPolicy = Object.assign({}, policy);
    clientPolicy.deployments =
        whitelists && whitelists.length
            ? whitelists
                  .filter(o => o.deployment !== undefined && o.deployment.name !== undefined)
                  .map(o => o.deployment.name)
            : [];
    clientPolicy.images =
        whitelists && whitelists.length
            ? whitelists
                  .filter(o => o.image !== undefined && o.image.name !== undefined)
                  .map(o => o.image.name)
            : [];
    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    serverPolicy.whitelists = [];
    if (policy.deployments && policy.deployments.length !== 0) {
        serverPolicy.whitelists = policy.deployments.map(name => ({ deployment: { name } }));
    }
    if (policy.images && policy.images.length !== 0) {
        serverPolicy.whitelists = serverPolicy.whitelists.concat(
            policy.images.map(name => ({ image: { name } }))
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
