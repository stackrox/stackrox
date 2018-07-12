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

export function preFormatCapabilitiesField(policy) {
    const newPolicy = Object.assign({}, policy);
    const { fields } = newPolicy;
    if (fields && fields.addCapabilities && fields.addCapabilities.length !== 0)
        fields.addCapabilities = fields.addCapabilities.map(o => ({ label: o, value: o }));
    if (fields && fields.dropCapabilities && fields.dropCapabilities.length !== 0)
        fields.dropCapabilities = fields.dropCapabilities.map(o => ({ label: o, value: o }));
    newPolicy.fields = fields;
    return newPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    serverPolicy.whitelists = [];
    if (policy.deployments && policy.deployments.length !== 0) {
        serverPolicy.whitelists = policy.deployments.map(o => ({ deployment: { name: o.label } }));
    }

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

export function postFormatCapabilitiesField(policy) {
    const serverPolicy = Object.assign({}, policy);
    const { fields } = serverPolicy;
    if (fields && fields.addCapabilities && fields.addCapabilities.length !== 0)
        fields.addCapabilities = fields.addCapabilities.map(o => o.value);
    if (fields && fields.dropCapabilities && fields.dropCapabilities.length !== 0)
        fields.dropCapabilities = fields.dropCapabilities.map(o => o.value);
    serverPolicy.fields = fields;
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
        value =>
            value === null ||
            value === undefined ||
            value === '' ||
            value === [] ||
            (Array.isArray(value) && !value.length)
    );
    const newObj = flatten.unflatten(omittedObj);
    return newObj;
}

export function preFormatPolicyFields(policy) {
    let formattedPolicy = removeEmptyFields(policy);
    formattedPolicy = preFormatWhitelistField(formattedPolicy);
    formattedPolicy = preFormatScopeField(formattedPolicy);
    formattedPolicy = preFormatCategoriesField(formattedPolicy);
    formattedPolicy = preFormatCapabilitiesField(formattedPolicy);
    return formattedPolicy;
}

export function formatPolicyFields(policy) {
    let serverPolicy = removeEmptyFields(policy);
    serverPolicy = postFormatWhitelistField(serverPolicy);
    serverPolicy = postFormatScopeField(serverPolicy);
    serverPolicy = postFormatCategoriesField(serverPolicy);
    serverPolicy = postFormatCapabilitiesField(serverPolicy);
    serverPolicy = postFormatNotifiersField(serverPolicy);
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
