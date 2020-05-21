export const comparatorOp = {
    GREATER_THAN: '>',
    GREATER_THAN_OR_EQUALS: '>=',
    EQUALS: '=',
    LESS_THAN_OR_EQUALS: '<=',
    LESS_THAN: '<',
};

export const formatResourceValue = (prefix, value, suffix) =>
    `${prefix} ${comparatorOp[value.op]} ${value.value} ${suffix}`;

export const formatResources = (resource) => {
    const output = [];
    if (resource.memoryResourceRequest) {
        output.push(formatResourceValue('Memory request', resource.memoryResourceRequest, 'MB'));
    }
    if (resource.memoryResourceLimit) {
        output.push(formatResourceValue('Memory limit', resource.memoryResourceLimit, 'MB'));
    }
    if (resource.cpuResourceRequest) {
        output.push(formatResourceValue('CPU request', resource.cpuResourceRequest, 'Cores'));
    }
    if (resource.cpuResourceLimit) {
        output.push(formatResourceValue('CPU limit', resource.cpuResourceLimit, 'Cores'));
    }
    return output.join(', ');
};

export const formatScope = (scope, props) => {
    if (!scope) return '';
    const values = [];
    if (scope.cluster && scope.cluster !== '') {
        let { cluster } = scope;
        if (props?.clustersById[scope.cluster]) {
            cluster = props.clustersById[scope.cluster].name;
        }
        values.push(`Cluster:${cluster}`);
    }
    if (scope.namespace && scope.namespace !== '') {
        values.push(`Namespace:${scope.namespace}`);
    }
    if (scope.label) {
        values.push(`Label:${scope.label.key}=${scope.label.value}`);
    }
    return values.join('; ');
};

export const formatDeploymentWhitelistScope = (whitelistScope, props) => {
    const values = [];
    if (whitelistScope.name && whitelistScope.name !== '') {
        values.push(`Deployment Name:${whitelistScope.name}`);
    }
    const scopeVal = formatScope(whitelistScope.scope, props);
    if (scopeVal !== '') {
        values.push(scopeVal);
    }
    return values.join('; ');
};
