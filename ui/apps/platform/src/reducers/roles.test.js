import reducer, { selectors, getHasReadPermission, getHasReadWritePermission } from './roles';

describe('roles reducer', () => {
    it('should return the initial state', () => {
        const expected = {
            roles: [],
            resources: [],
            selectedRole: null,
            userRolePermissions: null,
        };
        const state = reducer(undefined, {});

        expect(state).toEqual(expected);
    });
});

describe('userRolePermissions selector', () => {
    const { getUserRolePermissions } = selectors;

    it('should get the property from initial state', () => {
        const expected = null;
        const state = reducer(undefined, {});
        const received = getUserRolePermissions(state);

        expect(received).toEqual(expected);
    });

    it('should get the property from partial state', () => {
        const expected = {
            name: '',
            globalAccess: 'READ_WRITE_ACCESS',
            userRolePermissions: {
                Licenses: 'READ_ACCESS',
                ServiceIdentity: 'NO_ACCESS',
            },
        };
        const state = {
            userRolePermissions: expected,
        };
        const received = getUserRolePermissions(state);

        expect(received).toEqual(expected);
    });
});

describe('getHasReadPermission', () => {
    const permission = 'Licenses';

    it('should not have access given the initial state', () => {
        const state = null;
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if user role has no access', () => {
        const state = {
            name: '',
            globalAccess: 'READ_ACCESS',
            resourceToAccess: {
                Licenses: 'NO_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should have access if no user roles but global read access', () => {
        const state = {
            name: '',
            globalAccess: 'READ_ACCESS',
            resourceToAccess: null,
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should have access if user role has read access', () => {
        const state = {
            name: '',
            globalAccess: 'NO_ACCESS',
            resourceToAccess: {
                Licenses: 'READ_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should have access if user role has read-write access', () => {
        const state = {
            name: '',
            globalAccess: 'NO_ACCESS',
            resourceToAccess: {
                Licenses: 'READ_WRITE_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(true);
    });
});

describe('getHasReadWritePermission', () => {
    const permission = 'Licenses';

    it('should not have access given the initial state', () => {
        const state = null;
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if user role has no access', () => {
        const state = {
            name: '',
            globalAccess: 'READ_ACCESS',
            resourceToAccess: {
                Licenses: 'NO_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should have access if no user roles but global read-write access', () => {
        const state = {
            name: '',
            globalAccess: 'READ_WRITE_ACCESS',
            resourceToAccess: null,
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should have access if user role has read-write access', () => {
        const state = {
            name: '',
            globalAccess: 'NO_ACCESS',
            resourceToAccess: {
                Licenses: 'READ_WRITE_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should not have access if user role has read access', () => {
        const state = {
            name: '',
            globalAccess: 'READ_WRITE_ACCESS',
            resourceToAccess: {
                Licenses: 'READ_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });
});
