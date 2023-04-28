import { getFilteredSecurityContextMap } from './securityContextUtils';

describe('securityContextUtils', () => {
    describe('getFilteredSecurityContextMap', () => {
        it('should return an empty Map when there all security context values are falsy', () => {
            const securityContext = {
                privileged: false,
                selinux: null,
                dropCapabilities: [],
                addCapabilities: [],
                readOnlyRootFilesystem: false,
                seccompProfile: null,
                allowPrivilegeEscalation: false,
            };

            const filteredValues = getFilteredSecurityContextMap(securityContext);

            expect(filteredValues.size).toEqual(0);
        });

        it('should return a Map of only those security context values which are set', () => {
            const securityContext = {
                privileged: false,
                selinux: null,
                dropCapabilities: [],
                addCapabilities: [],
                readOnlyRootFilesystem: true,
                seccompProfile: null,
                allowPrivilegeEscalation: false,
            };

            const filteredValues = getFilteredSecurityContextMap(securityContext);

            const expectedMap = new Map();
            expectedMap.set('readOnlyRootFilesystem', 'true');

            expect(filteredValues).toEqual(expectedMap);
        });

        it('should return a Map which includes array and object security context values which are set', () => {
            const securityContext = {
                privileged: false,
                selinux: {
                    user: '',
                    role: '',
                    type: 'container_runtime_t',
                    level: '',
                },
                dropCapabilities: ['NET_RAW', 'CAP_SYS_TIME'],
                addCapabilities: [],
                readOnlyRootFilesystem: true,
                seccompProfile: null,
                allowPrivilegeEscalation: false,
            };

            const filteredValues = getFilteredSecurityContextMap(securityContext);

            const expectedMap = new Map();
            expectedMap.set('readOnlyRootFilesystem', 'true');
            expectedMap.set(
                'selinux',
                '{"user":"","role":"","type":"container_runtime_t","level":""}'
            );
            expectedMap.set('dropCapabilities', 'NET_RAW,CAP_SYS_TIME');

            expect(filteredValues).toEqual(expectedMap);
        });
    });
});
