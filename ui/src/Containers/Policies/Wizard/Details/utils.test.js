import {
    formatResourceValue,
    formatResources,
    formatScope,
    formatDeploymentWhitelistScope,
} from './utils';

describe('policyDetailsUtils', () => {
    describe('formatResourceValue', () => {
        it('should add comparator string in appropriate spot in string', () => {
            const prefix = 'Memory limit';
            const value = {
                op: 'EQUALS',
                value: '3000',
            };
            const suffix = 'MB';
            const formattedString = formatResourceValue(prefix, value, suffix);
            expect(formattedString).toEqual('Memory limit = 3000 MB');
        });
    });

    describe('formatResources', () => {
        it('should format resources into string based on resource', () => {
            const resource = {
                memoryResourceLimit: {
                    op: 'EQUALS',
                    value: '3000',
                },
            };
            const valueStr = formatResources(resource);
            expect(valueStr).toBe('Memory limit = 3000 MB');
        });
        it('should not format anything if resource array is empty', () => {
            const resource = {};
            const valueStr = formatResources(resource);
            expect(valueStr).toBe('');
        });
        it('should append multiple resources if there are multiple resources in array', () => {
            const resource = {
                memoryResourceLimit: {
                    op: 'EQUALS',
                    value: '3000',
                },
                cpuResourceLimit: {
                    op: 'LESS_THAN',
                    value: '2',
                },
            };
            const valueStr = formatResources(resource);
            expect(valueStr).toBe('Memory limit = 3000 MB, CPU limit < 2 Cores');
        });
    });
    describe('formatScope', () => {
        it('should return empty string if no scope is defined', () => {
            const scope = null;
            const valueStr = formatScope(scope);
            expect(valueStr).toBe('');
        });
        it('should format cluster scope if cluster is defined', () => {
            const scope = {
                cluster: 'remote',
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe(`Cluster:${scope.cluster}`);
        });
        it('should not format cluster scope if cluster is not defined', () => {
            const scope = {
                cluster: undefined,
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe('');
        });
        it('should format namespace scope if namespace is defined', () => {
            const scope = {
                namespace: 'kube-system',
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe(`Namespace:${scope.namespace}`);
        });
        it('should not format namespace scope if namespace is not defined', () => {
            const scope = {
                namespace: undefined,
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe('');
        });
        it('should format label scope if label is defined', () => {
            const scope = {
                label: {
                    key: 'key',
                    value: 'value',
                },
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe(`Label:${scope.label.key}=${scope.label.value}`);
        });
        it('should not format label scope if label is not defined', () => {
            const scope = {
                label: undefined,
            };
            const valueStr = formatScope(scope);
            expect(valueStr).toBe('');
        });
    });
    describe('formatDeploymentWhitelistScope', () => {
        it('should format deployment scope if deployment is defined', () => {
            const whitelistScope = {
                name: 'nginx',
            };
            const valueStr = formatDeploymentWhitelistScope(whitelistScope);
            expect(valueStr).toBe(`Deployment Name:${whitelistScope.name}`);
        });
        it('should not format deployment scope if deployment is not defined', () => {
            const whitelistScope = {
                name: undefined,
            };
            const valueStr = formatDeploymentWhitelistScope(whitelistScope);
            expect(valueStr).toBe('');
        });
    });
});
