import policyFormFields from 'Containers/Policies/policyCreationFormDescriptor';
import removeEmptyFields from 'utils/removeEmptyFields';

export function preFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    newObj.scope = obj.scope && obj.scope.length !== 0 ? obj.scope.map(o => o.cluster) : [];
    return newObj;
}

export function postFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    if (newObj.scope && newObj.scope.length !== 0) {
        newObj.scope = obj.scope.map(clusterId => ({ cluster: clusterId }));
    }
    return newObj;
}

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    const clientPolicy = Object.assign({}, policy);
    clientPolicy.deployments =
        whitelists && whitelists.length
            ? whitelists.filter(o => o.deployment.name !== undefined).map(o => o.deployment.name)
            : [];
    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    serverPolicy.whitelists = [];
    if (policy.deployments && policy.deployments.length !== 0) {
        serverPolicy.whitelists = policy.deployments.map(name => ({ deployment: { name } }));
    }

    return serverPolicy;
}

export function preFormatPolicyFields(policy) {
    let formattedPolicy = removeEmptyFields(policy);
    formattedPolicy = preFormatWhitelistField(formattedPolicy);
    formattedPolicy = preFormatScopeField(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyFields(policy);
    serverPolicy = postFormatWhitelistField(serverPolicy);
    serverPolicy = postFormatScopeField(serverPolicy);
    return serverPolicy;
}

export function mapDescriptorToKey(descriptor) {
    return descriptor.map(obj => obj.jsonpath);
}

export function getPolicyFormDataKeys() {
    const { policyDetails, policyConfiguration } = policyFormFields;
    const policyDetailsKeys = mapDescriptorToKey(policyDetails.descriptor);
    const policyConfigurationKeys = mapDescriptorToKey(policyConfiguration.descriptor);
    return [...policyDetailsKeys, ...policyConfigurationKeys];
}
