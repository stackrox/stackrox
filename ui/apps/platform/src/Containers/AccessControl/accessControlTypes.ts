import { ReactNode } from 'react';

// Type definitions, mock data, and helper functions that might be temporary.

export type Column = {
    Header: string;
    accessor: string;
    Cell?: (CellCallbackArgs) => ReactNode;
    headerClassName: string;
    className: string;
    sortable?: boolean;
};

export type AccessScope = {
    id: string;
    name: string;
    description: string;
};

// TODO what is this and where is its endpoint and proto?
export type AuthProvider = {
    id: string;
    name: string;
    authProvider: string;
    minimumAccessRole: string;
    // assignedRules: string[];
};

export type Access = 'NO_ACCESS' | 'READ_ACCESS' | 'READ_WRITE_ACCESS';

export type Permission = {
    resource: string;
    access: Access;
};

export type PermissionSet = {
    id: string;
    name: string;
    description: string;
    minimumAccessLevel: string;
    permissions: Permission[];
};

export type Role = {
    id: string;
    name: string;
    description: string;
    permissionSetId: string;
    accessScopeId: string;
};

export type AccessControlEntity = AccessScope | AuthProvider | PermissionSet | Role;

interface HasId {
    id: string;
}

function getAccessControlMap<T extends HasId>(array: T[]): Record<string, T> {
    const map: Record<string, T> = {};

    array.forEach((item) => {
        map[item.id] = item;
    });

    return map;
}

// Mock data
export const authProviders: AuthProvider[] = [
    {
        id: '0',
        name: 'Read-Only Auth0',
        authProvider: 'Auth0',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '1',
        name: 'Read-Write OpenID',
        authProvider: 'OpenID Connect',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '2',
        name: 'SeriousSAML',
        authProvider: 'SAML 2.0',
        minimumAccessRole: 'None',
    },
];

// Mock data
export const roles: Role[] = [
    {
        id: 'GuestUser',
        name: 'Guest User',
        description: 'Has access to selected entities',
        permissionSetId: 'GuestAccount',
        accessScopeId: 'WalledGarden',
    },
    {
        id: 'Admin',
        name: 'Admin',
        description: 'Admin access to all entities',
        permissionSetId: 'ReadWriteAll',
        accessScopeId: 'AllAccess',
    },
    {
        id: 'SensorCreator',
        name: 'Sensor Creator',
        description: 'Users can create sensors',
        permissionSetId: 'WriteSpecific',
        accessScopeId: 'LimitedAccess',
    },
    {
        id: 'None',
        name: 'None',
        description: 'No access',
        permissionSetId: 'NoPermissions',
        accessScopeId: 'DenyAccess',
    },
    {
        id: 'ContinuousIntegration',
        name: 'Continuous Integration',
        description: 'Users can manage integrations',
        permissionSetId: 'WriteSpecific',
        accessScopeId: 'LimitedAccess',
    },
    {
        id: 'Analyst',
        name: 'Analyst',
        description: 'Users can view and create reports',
        permissionSetId: 'ReadOnly',
        accessScopeId: 'LimitedAccess',
    },
];

export const rolesMap = getAccessControlMap(roles);

// Mock data
export const permissionSets: PermissionSet[] = [
    {
        id: 'GuestAccount',
        name: 'Guest Account',
        description: 'Limited write access to basic settings, cannot save changes',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: 'ReadWriteAll',
        name: 'Read-Write All',
        description: 'Full read and write access',
        minimumAccessLevel: 'READ_WRITE_ACCESS',
        permissions: [],
    },
    {
        id: 'WriteSpecific',
        name: 'Write Specific',
        description: 'Limited write access and full read access',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: 'NoPermissions',
        name: 'No Permissions',
        description: 'No read or write access',
        minimumAccessLevel: 'NO_ACCESS',
        permissions: [],
    },
    {
        id: 'ReadOnly',
        name: 'Read Only',
        description: 'Full read access, no write access',
        minimumAccessLevel: 'READ_ACCESS',
        permissions: [],
    },
    {
        id: 'Test Set',
        name: 'TestSet',
        description: 'Experimental set, do not use',
        minimumAccessLevel: 'NO_ACCESS',
        permissions: [],
    },
];

export const permissionSetsMap = getAccessControlMap(permissionSets);

// Mock data
export const accessScopes: AccessScope[] = [
    {
        id: 'WalledGarden',
        name: 'Walled Garden',
        description: 'Exclude all entities, only access select entities',
    },
    {
        id: 'AllAccess',
        name: 'All Access',
        description: 'Users can access all entities',
    },
    {
        id: 'LimitedAccess',
        name: 'Limited Access',
        description: 'Users have access to limited entities',
    },
    {
        id: 'DenyAccess',
        name: 'Deny Access',
        description: 'Users have no access',
    },
];

export const accessScopesMap = getAccessControlMap(accessScopes);
