import { getFilteredSecurityContextMap } from './securityContextUtils';

describe('securityContextUtils', () => {
    describe('getFilteredSecurityContextMap', () => {
        it('should return an empty array when there all security context values are falsy', () => {
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

            expect(filteredValues.length).toEqual(0);
        });

        it('should return an array of only those security context values which are set', () => {
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

            expect(filteredValues).toEqual([['Read Only Root Filesystem', 'true']]);
        });

        it('should return an array which includes array and object security context values which are set', () => {
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

            expect(filteredValues).toEqual([
                ['Drop Capabilities', 'NET_RAW,CAP_SYS_TIME'],
                ['Read Only Root Filesystem', 'true'],
                ['Selinux', '{"user":"","role":"","type":"container_runtime_t","level":""}'],
            ]);
        });
    });
});
