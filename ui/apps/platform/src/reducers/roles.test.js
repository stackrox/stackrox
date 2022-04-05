import reducer, { selectors, getHasReadPermission, getHasReadWritePermission } from './roles';

describe('roles reducer', () => {
    it('should return the initial state', () => {
        const expected = {
            roles: [],
            resources: [],
            selectedRole: null,
            userRolePermissions: null,
            userRolePermissionsError: null,
            isLoadingUserRolePermissions: true,
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
            userRolePermissions: {
                Deployment: 'READ_ACCESS',
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
    const permission = 'Deployment';

    it('should not have access given the initial state', () => {
        const state = null;
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resource has no access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'NO_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resourceToAccess is null', () => {
        const state = {
            name: '',
            resourceToAccess: null,
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resourceToAccess does not have the resource', () => {
        const state = {
            name: '',
            resourceToAccess: {},
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should have access if resource has read access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'READ_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should have access if resource has read-write access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'READ_WRITE_ACCESS',
            },
        };
        const received = getHasReadPermission(permission, state);

        expect(received).toEqual(true);
    });
});

describe('getHasReadWritePermission', () => {
    const permission = 'Deployment';

    it('should not have access given the initial state', () => {
        const state = null;
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resource has no access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'NO_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resourceToAccess is null', () => {
        const state = {
            name: '',
            resourceToAccess: null,
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should not have access if resourceToAccess does not have the resource', () => {
        const state = {
            name: '',
            resourceToAccess: {},
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });

    it('should have access if resource has read-write access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'READ_WRITE_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(true);
    });

    it('should not have access if resource has read access', () => {
        const state = {
            name: '',
            resourceToAccess: {
                Deployment: 'READ_ACCESS',
            },
        };
        const received = getHasReadWritePermission(permission, state);

        expect(received).toEqual(false);
    });
});
