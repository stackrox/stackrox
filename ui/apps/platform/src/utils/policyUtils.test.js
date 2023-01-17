// system under test (SUT)
import { getExcludedNamesByType } from './policyUtils';

describe('policyUtils', () => {
    describe('getExcludedNamesByType', () => {
        it('should return an empty string when no scopes of the requested type are present', () => {
            const excludedScopes = [
                {
                    deployment: null,
                    expiration: null,
                    image: {
                        name: 'docker.io/library/mysql:5',
                    },
                    name: '',
                },
            ];

            const names = getExcludedNamesByType(excludedScopes, 'deployment');

            expect(names).toEqual('');
        });

        it('should return a list of only the excluded deployment names', () => {
            const excludedScopes = [
                {
                    deployment: {
                        name: 'central',
                        scope: null,
                    },
                    expiration: null,
                    image: null,
                    name: '',
                },
                {
                    deployment: {
                        name: 'kube-proxy',
                        scope: null,
                    },
                    expiration: null,
                    image: null,
                    name: '',
                },
                {
                    deployment: null,
                    expiration: null,
                    image: {
                        name: 'docker.io/library/mysql:5',
                    },
                    name: '',
                },
            ];

            const names = getExcludedNamesByType(excludedScopes, 'deployment');

            expect(names).toEqual('central, kube-proxy');
        });

        it('should return a list of only the excluded image names', () => {
            const excludedScopes = [
                {
                    deployment: {
                        name: 'central',
                        scope: null,
                    },
                    expiration: null,
                    image: null,
                    name: '',
                },
                {
                    deployment: null,
                    expiration: null,
                    image: {
                        name: 'docker.io/library/mysql:5',
                    },
                    name: '',
                },
                {
                    deployment: null,
                    expiration: null,
                    image: {
                        name: 'registry.k8s.io/coredns:1.3.1',
                    },
                    name: '',
                },
            ];

            const names = getExcludedNamesByType(excludedScopes, 'image');

            expect(names).toEqual('docker.io/library/mysql:5, registry.k8s.io/coredns:1.3.1');
        });
    });
});
