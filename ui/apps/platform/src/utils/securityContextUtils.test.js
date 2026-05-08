import { getFilteredSecurityContextMap } from './securityContextUtils';

describe('securityContextUtils', () => {
    describe('getFilteredSecurityContextMap', () => {
        it('should return an empty array when all security context values are falsy', () => {
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

            expect(filteredValues).toEqual([['Read only root filesystem', 'true']]);
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
                ['Drop capabilities', 'NET_RAW,CAP_SYS_TIME'],
                ['Read only root filesystem', 'true'],
                ['SELinux', '{"user":"","role":"","type":"container_runtime_t","level":""}'],
            ]);
        });

        it('should map every known key to a sentence-case label when all are set', () => {
            const seccomp = { type: 'UNCONFINED', localhostProfile: '' };
            const selinux = {
                user: 'u',
                role: 'r',
                type: 't',
                level: 'l',
            };

            const filteredValues = getFilteredSecurityContextMap({
                privileged: true,
                selinux,
                dropCapabilities: ['NET_RAW'],
                addCapabilities: ['NET_ADMIN'],
                readOnlyRootFilesystem: true,
                seccompProfile: seccomp,
                allowPrivilegeEscalation: true,
            });

            expect(filteredValues).toEqual([
                ['Add capabilities', 'NET_ADMIN'],
                ['Allow privilege escalation', 'true'],
                ['Drop capabilities', 'NET_RAW'],
                ['Privileged', 'true'],
                ['Read only root filesystem', 'true'],
                ['Seccomp profile', JSON.stringify(seccomp)],
                ['SELinux', JSON.stringify(selinux)],
            ]);
        });

        it('should use the raw key as the label for unknown future fields', () => {
            const withExtra = {
                privileged: true,
                selinux: null,
                dropCapabilities: [],
                addCapabilities: [],
                readOnlyRootFilesystem: false,
                seccompProfile: null,
                allowPrivilegeEscalation: false,
                futureFieldName: 1,
            };

            const filteredValues = getFilteredSecurityContextMap(withExtra);

            expect(filteredValues).toEqual([
                ['futureFieldName', '1'],
                ['Privileged', 'true'],
            ]);
        });
    });
});
