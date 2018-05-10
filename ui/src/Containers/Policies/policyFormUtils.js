import flatten from 'flat';
import omitBy from 'lodash/omitBy';

export function preFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    if (obj.scope) newObj.scope = obj.scope.map(o => o.cluster);
    return newObj;
}

export function postFormatScopeField(obj) {
    const newObj = Object.assign({}, obj);
    if (newObj.scope) newObj.scope = obj.scope.map(o => ({ cluster: o }));
    return newObj;
}

export function preFormatWhitelistField(policy) {
    const { whitelists } = policy;
    if (!whitelists || !whitelists.length) {
        return policy;
    }
    const clientPolicy = Object.assign({}, policy);
    clientPolicy.deployments = whitelists
        .filter(o => o.deployment.name !== undefined)
        .map(o => o.deployment.name);
    return clientPolicy;
}

export function postFormatWhitelistField(policy) {
    const serverPolicy = Object.assign({}, policy);
    if (policy.deployments)
        serverPolicy.whitelists = policy.deployments.map(o => ({ deployment: { name: o } }));
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
