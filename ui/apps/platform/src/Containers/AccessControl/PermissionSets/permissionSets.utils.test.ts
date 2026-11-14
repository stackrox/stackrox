import type { PermissionSet } from 'services/RolesService';

import { setPermissionSetConstraint } from './permissionSets.utils';

const permissionSet: PermissionSet = {
    id: 'example',
    name: 'Example',
    description: '',
    resourceToAccess: {
        Administration: 'READ_WRITE_ACCESS',
        Deployment: 'READ_WRITE_ACCESS',
        Node: 'READ_ACCESS',
    },
};

describe('setPermissionSetConstraint', () => {
    it('preserves reads when enabling read-access-only', () => {
        const result = setPermissionSetConstraint(permissionSet, 'accessConstraint', true, [
            'Administration',
        ]);
        expect(result.resourceToAccess).toEqual({
            Administration: 'READ_ACCESS',
            Deployment: 'READ_ACCESS',
            Node: 'READ_ACCESS',
        });
        expect(result.accessConstraint).toBe('ACCESS_CONSTRAINT_READ');
        expect(permissionSet.resourceToAccess.Deployment).toBe('READ_WRITE_ACCESS');
    });

    it('removes access to global resources when enabling scoped-only', () => {
        const result = setPermissionSetConstraint(permissionSet, 'scopeConstraint', true, [
            'Administration',
        ]);
        expect(result.resourceToAccess).toEqual({
            Administration: 'NO_ACCESS',
            Deployment: 'READ_WRITE_ACCESS',
            Node: 'READ_ACCESS',
        });
        expect(result.scopeConstraint).toBe('SCOPE_CONSTRAINT_CLUSTER_NAMESPACE');
    });

    it('does not restore removed grants when disabling either constraint', () => {
        const readAccessOnly = setPermissionSetConstraint(
            permissionSet,
            'accessConstraint',
            true,
            []
        );
        const result = setPermissionSetConstraint(readAccessOnly, 'accessConstraint', false, []);
        expect(result.resourceToAccess.Deployment).toBe('READ_ACCESS');
        expect(result.accessConstraint).toBe('ACCESS_CONSTRAINT_UNSPECIFIED');

        const scopedOnly = setPermissionSetConstraint(permissionSet, 'scopeConstraint', true, [
            'Administration',
        ]);
        const unscoped = setPermissionSetConstraint(scopedOnly, 'scopeConstraint', false, [
            'Administration',
        ]);
        expect(unscoped.resourceToAccess.Administration).toBe('NO_ACCESS');
        expect(unscoped.scopeConstraint).toBe('SCOPE_CONSTRAINT_UNSPECIFIED');
    });
});
