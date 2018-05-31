import flatten from 'flat';
import omitBy from 'lodash/omitBy';
import policyFormFields from 'Containers/Policies/policyCreationFormDescriptor';

export function preFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    newObj.scope = obj.scope && obj.scope.length !== 0 ? obj.scope.map(o => o.cluster) : [];
    return newObj;
}

export function postFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    if (newObj.scope && newObj.scope.length !== 0) {
        newObj.scope = obj.scope.map(
            cluster => (typeof cluster === 'object' ? { cluster: cluster.value } : { cluster })
        );
    }
    return newObj;
}

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    const clientPolicy = Object.assign({}, policy);
    clientPolicy.deployments =
        whitelists && whitelists.length
            ? whitelists
                  .filter(o => o.deployment.name !== undefined)
                  .map(o => ({ label: o.deployment.name, value: o.deployment.name }))
            : [];
    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    if (policy.deployments && policy.deployments.length !== 0)
        serverPolicy.whitelists = policy.deployments.map(o => ({ deployment: { name: o.label } }));
    return serverPolicy;
}

export function preFormatCategoriesField(policy) {
    const serverPolicy = Object.assign({}, policy);
    serverPolicy.categories =
        policy.categories && policy.categories.length !== 0
            ? policy.categories.map(category => ({ label: category, value: category }))
            : [];
    return serverPolicy;
}

export function postFormatCategoriesField(policy) {
    const serverPolicy = Object.assign({}, policy);
    if (policy.categories && policy.categories.length !== 0)
        serverPolicy.categories = policy.categories.map(o => o.value);
    return serverPolicy;
}

export function postFormatNotifiersField(policy) {
    const serverPolicy = Object.assign({}, policy);
    if (policy.notifiers && policy.notifiers.length !== 0)
        serverPolicy.notifiers = policy.notifiers.map(o => (o.value ? o.value : o));
    return serverPolicy;
}

export function removeEmptyFields(obj) {
    const flattenedObj = flatten(obj);
    const omittedObj = omitBy(
        flattenedObj,
        value => value === null || value === undefined || value === '' || value === []
    );
    const newObj = flatten.unflatten(omittedObj);
    return newObj;
}

export function preFormatPolicyFields(policy) {
    let formattedPolicy = removeEmptyFields(policy);
    formattedPolicy = preFormatWhitelistField(formattedPolicy);
    formattedPolicy = preFormatScopeField(formattedPolicy);
    formattedPolicy = preFormatCategoriesField(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyFields(policy);
    serverPolicy = postFormatWhitelistField(serverPolicy);
    serverPolicy = postFormatScopeField(serverPolicy);
    serverPolicy = postFormatCategoriesField(serverPolicy);
    serverPolicy = postFormatNotifiersField(serverPolicy);
    return serverPolicy;
}

export function mapDescriptorToKey(descriptor) {
    return descriptor.map(obj => obj.jsonpath);
}

export function getPolicyFormDataKeys() {
    const { policyDetails, imagePolicy, configurationPolicy, privilegePolicy } = policyFormFields;
    const policyDetailsKeys = mapDescriptorToKey(policyDetails.descriptor);
    const imagePolicyKeys = mapDescriptorToKey(imagePolicy.descriptor);
    const configurationPolicyKeys = mapDescriptorToKey(configurationPolicy.descriptor);
    const privilegePolicyKeys = mapDescriptorToKey(privilegePolicy.descriptor);
    return policyDetailsKeys.concat(imagePolicyKeys, configurationPolicyKeys, privilegePolicyKeys);
}
