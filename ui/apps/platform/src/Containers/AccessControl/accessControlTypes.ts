export type Column = {
    Header: string;
    accessor: string;
    headerClassName: string;
    className: string;
    sortable?: boolean;
};

type Type = 'User defined' | 'System default';

export type AccessScope = {
    id: string;
    name: string;
    description: string;
    type: Type; // not yet in proto
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
    displayName: string;
    description: string;
    type: Type; // not yet in proto
    minimumAccessLevel: Access;
    permissions: Permission[];
};

export type Role = {
    id: string;
    name: string;
    displayName: string;
    description: string;
    type: Type; // not yet in proto
    permissionSetName: string;
    accessScopeName: string;
};

export type AccessControlRow = AccessScope | AuthProvider | PermissionSet | Role;
