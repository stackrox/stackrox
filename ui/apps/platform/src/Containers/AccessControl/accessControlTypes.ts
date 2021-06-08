import { AccessControlEntityType } from 'constants/entityTypes';
import { AccessType as AccessLevel } from 'services/RolesService';

export type AccessControlQueryAction = 'create' | 'update';

export type AccessControlQueryObject = {
    action?: AccessControlQueryAction;
    s?: Partial<Record<AccessControlEntityType, string>>;
};

// Type definitions, mock data, and helper functions that might be temporary.

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

export type Permission = {
    resource: string;
    access: AccessLevel;
};

export type PermissionSet = {
    id: string;
    name: string;
    description: string;
    minimumAccessLevel: AccessLevel;
    resourceIdToAccess: Record<string, AccessLevel>;
};

export type Role = {
    id: string;
    name: string;
    description: string;
    permissionSetId: string;
    accessScopeId: string;
};

export type AccessControlEntity = AccessScope | AuthProvider | PermissionSet | Role;

const delay = 1000;

// Mock data and requests
export const authProviders: AuthProvider[] = [
    {
        id: '0',
        name: 'Read-Only Auth0',
        authProvider: 'auth0',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '1',
        name: 'Read-Write OpenID',
        authProvider: 'oidc',
        minimumAccessRole: 'Analyst',
    },
    {
        id: '2',
        name: 'SeriousSAML',
        authProvider: 'saml',
        minimumAccessRole: 'None',
    },
];

export function fetchAuthProviders(): Promise<AuthProvider[]> {
    // Thank you, Stack Overflow :)
    return Promise.resolve(authProviders).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

let idAuthProvider = 3;

export function createAuthProvider(values: AuthProvider): Promise<AuthProvider> {
    // Thank you, Stack Overflow :)
    // eslint-disable-next-line no-plusplus
    return Promise.resolve({ ...values, id: String(idAuthProvider++) }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

export function updateAuthProvider(values: AuthProvider): Promise<AuthProvider> {
    // Thank you, Stack Overflow :)
    return Promise.resolve({ ...values }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

// Mock data and requests
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

export function fetchRoles(): Promise<Role[]> {
    // Thank you, Stack Overflow :)
    return Promise.resolve(roles).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

let idRole = 0;

export function createRole(values: Role): Promise<Role> {
    // Thank you, Stack Overflow :)
    // eslint-disable-next-line no-plusplus
    return Promise.resolve({ ...values, id: String(idRole++) }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

export function updateRole(values: Role): Promise<Role> {
    // Thank you, Stack Overflow :)
    return Promise.resolve({ ...values }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

// Mock data and requests
export const permissionSets: PermissionSet[] = [
    {
        id: 'ContinuousIntegration',
        name: 'Continuous Integration',
        description: 'Has read-only access to check images and deployment YAMLs against policies',
        minimumAccessLevel: 'NO_ACCESS',
        resourceIdToAccess: {
            Detection: 'READ_ACCESS',
            Image: 'READ_WRITE_ACCESS',
        },
    },
    {
        id: 'GuestAccount',
        name: 'Guest Account',
        description: 'Limited write access to basic settings, cannot save changes',
        minimumAccessLevel: 'READ_ACCESS',
        resourceIdToAccess: {},
    },
    {
        id: 'ReadWriteAll',
        name: 'Read-Write All',
        description: 'Full read and write access',
        minimumAccessLevel: 'READ_WRITE_ACCESS',
        resourceIdToAccess: {
            APIToken: 'READ_WRITE_ACCESS',
            Alert: 'READ_WRITE_ACCESS',
            AllComments: 'READ_WRITE_ACCESS',
            AuthPlugin: 'READ_WRITE_ACCESS',
            AuthProvider: 'READ_WRITE_ACCESS',
            BackupPlugins: 'READ_WRITE_ACCESS',
            CVE: 'READ_WRITE_ACCESS',
            Cluster: 'READ_WRITE_ACCESS',
            Compliance: 'READ_WRITE_ACCESS',
            ComplianceRunSchedule: 'READ_WRITE_ACCESS',
            ComplianceRuns: 'READ_WRITE_ACCESS',
            Config: 'READ_WRITE_ACCESS',
            DebugLogs: 'READ_WRITE_ACCESS',
            Deployment: 'READ_WRITE_ACCESS',
            Detection: 'READ_WRITE_ACCESS',
            Group: 'READ_WRITE_ACCESS',
            Image: 'READ_WRITE_ACCESS',
            ImageComponent: 'READ_WRITE_ACCESS',
            ImageIntegration: 'READ_WRITE_ACCESS',
            Indicator: 'READ_WRITE_ACCESS',
            K8sRole: 'READ_WRITE_ACCESS',
            K8sRoleBinding: 'READ_WRITE_ACCESS',
            K8sSubject: 'READ_WRITE_ACCESS',
            Licenses: 'READ_WRITE_ACCESS',
            LogIntegration: 'READ_WRITE_ACCESS',
            Namespace: 'READ_WRITE_ACCESS',
            NetworkBaseline: 'READ_WRITE_ACCESS',
            NetworkGraph: 'READ_WRITE_ACCESS',
            NetworkGraphConfig: 'READ_WRITE_ACCESS',
            NetworkPolicy: 'READ_WRITE_ACCESS',
            Node: 'READ_WRITE_ACCESS',
            Notifier: 'READ_WRITE_ACCESS',
            Policy: 'READ_WRITE_ACCESS',
            ProbeUpload: 'READ_WRITE_ACCESS',
            ProcessWhitelist: 'READ_WRITE_ACCESS',
            Risk: 'READ_WRITE_ACCESS',
            Role: 'READ_WRITE_ACCESS',
            ScannerBundle: 'READ_WRITE_ACCESS',
            ScannerDefinitions: 'READ_WRITE_ACCESS',
            Secret: 'READ_WRITE_ACCESS',
            SensorUpgradeConfig: 'READ_WRITE_ACCESS',
            ServiceAccount: 'READ_WRITE_ACCESS',
            ServiceIdentity: 'READ_WRITE_ACCESS',
            User: 'READ_WRITE_ACCESS',
            WatchedImage: 'READ_WRITE_ACCESS',
        },
    },
    {
        id: 'WriteSpecific',
        name: 'Write Specific',
        description: 'Limited write access and full read access',
        minimumAccessLevel: 'READ_ACCESS',
        resourceIdToAccess: {},
    },
    {
        id: 'NoPermissions',
        name: 'No Permissions',
        description: 'No read or write access',
        minimumAccessLevel: 'NO_ACCESS',
        resourceIdToAccess: {
            APIToken: 'NO_ACCESS',
            Alert: 'NO_ACCESS',
            AllComments: 'NO_ACCESS',
            AuthPlugin: 'NO_ACCESS',
            AuthProvider: 'NO_ACCESS',
            BackupPlugins: 'NO_ACCESS',
            CVE: 'NO_ACCESS',
            Cluster: 'NO_ACCESS',
            Compliance: 'NO_ACCESS',
            ComplianceRunSchedule: 'NO_ACCESS',
            ComplianceRuns: 'NO_ACCESS',
            Config: 'NO_ACCESS',
            DebugLogs: 'NO_ACCESS',
            Deployment: 'NO_ACCESS',
            Detection: 'NO_ACCESS',
            Group: 'NO_ACCESS',
            Image: 'NO_ACCESS',
            ImageComponent: 'NO_ACCESS',
            ImageIntegration: 'NO_ACCESS',
            Indicator: 'NO_ACCESS',
            K8sRole: 'NO_ACCESS',
            K8sRoleBinding: 'NO_ACCESS',
            K8sSubject: 'NO_ACCESS',
            Licenses: 'NO_ACCESS',
            LogIntegration: 'NO_ACCESS',
            Namespace: 'NO_ACCESS',
            NetworkBaseline: 'NO_ACCESS',
            NetworkGraph: 'NO_ACCESS',
            NetworkGraphConfig: 'NO_ACCESS',
            NetworkPolicy: 'NO_ACCESS',
            Node: 'NO_ACCESS',
            Notifier: 'NO_ACCESS',
            Policy: 'NO_ACCESS',
            ProbeUpload: 'NO_ACCESS',
            ProcessWhitelist: 'NO_ACCESS',
            Risk: 'NO_ACCESS',
            Role: 'NO_ACCESS',
            ScannerBundle: 'NO_ACCESS',
            ScannerDefinitions: 'NO_ACCESS',
            Secret: 'NO_ACCESS',
            SensorUpgradeConfig: 'NO_ACCESS',
            ServiceAccount: 'NO_ACCESS',
            ServiceIdentity: 'NO_ACCESS',
            User: 'NO_ACCESS',
            WatchedImage: 'NO_ACCESS',
        },
    },
    {
        id: 'ReadOnly',
        name: 'Read Only',
        description: 'Full read access, no write access',
        minimumAccessLevel: 'READ_ACCESS',
        resourceIdToAccess: {
            APIToken: 'READ_ACCESS',
            Alert: 'READ_ACCESS',
            AllComments: 'READ_ACCESS',
            AuthPlugin: 'READ_ACCESS',
            AuthProvider: 'READ_ACCESS',
            BackupPlugins: 'READ_ACCESS',
            CVE: 'READ_ACCESS',
            Cluster: 'READ_ACCESS',
            Compliance: 'READ_ACCESS',
            ComplianceRunSchedule: 'READ_ACCESS',
            ComplianceRuns: 'READ_ACCESS',
            Config: 'READ_ACCESS',
            DebugLogs: 'READ_ACCESS',
            Deployment: 'READ_ACCESS',
            Detection: 'READ_ACCESS',
            Group: 'READ_ACCESS',
            Image: 'READ_ACCESS',
            ImageComponent: 'READ_ACCESS',
            ImageIntegration: 'READ_ACCESS',
            Indicator: 'READ_ACCESS',
            K8sRole: 'READ_ACCESS',
            K8sRoleBinding: 'READ_ACCESS',
            K8sSubject: 'READ_ACCESS',
            Licenses: 'READ_ACCESS',
            LogIntegration: 'READ_ACCESS',
            Namespace: 'READ_ACCESS',
            NetworkBaseline: 'READ_ACCESS',
            NetworkGraph: 'READ_ACCESS',
            NetworkGraphConfig: 'READ_ACCESS',
            NetworkPolicy: 'READ_ACCESS',
            Node: 'READ_ACCESS',
            Notifier: 'READ_ACCESS',
            Policy: 'READ_ACCESS',
            ProbeUpload: 'READ_ACCESS',
            ProcessWhitelist: 'READ_ACCESS',
            Risk: 'READ_ACCESS',
            Role: 'READ_ACCESS',
            ScannerBundle: 'READ_ACCESS',
            ScannerDefinitions: 'READ_ACCESS',
            Secret: 'READ_ACCESS',
            SensorUpgradeConfig: 'READ_ACCESS',
            ServiceAccount: 'READ_ACCESS',
            ServiceIdentity: 'READ_ACCESS',
            User: 'READ_ACCESS',
            WatchedImage: 'READ_ACCESS',
        },
    },
    {
        id: 'TestSet',
        name: 'Test Set',
        description: 'Experimental set, do not use',
        minimumAccessLevel: 'NO_ACCESS',
        resourceIdToAccess: {},
    },
];

export function fetchPermissionSets(): Promise<PermissionSet[]> {
    // Thank you, Stack Overflow :)
    return Promise.resolve(permissionSets).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

let idPermissionSet = 0;

export function createPermissionSet(values: PermissionSet): Promise<PermissionSet> {
    // Thank you, Stack Overflow :)
    // eslint-disable-next-line no-plusplus
    return Promise.resolve({ ...values, id: String(idPermissionSet++) }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

export function updatePermissionSet(values: PermissionSet): Promise<PermissionSet> {
    // Thank you, Stack Overflow :)
    return Promise.resolve({ ...values }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

// Mock data and requests
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

export function fetchAccessScopes(): Promise<AccessScope[]> {
    // Thank you, Stack Overflow :)
    return Promise.resolve(accessScopes).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

let idAccessScope = 0;

export function createAccessScope(values: AccessScope): Promise<AccessScope> {
    // Thank you, Stack Overflow :)
    // eslint-disable-next-line no-plusplus
    return Promise.resolve({ ...values, id: String(idAccessScope++) }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}

export function updateAccessScope(values: AccessScope): Promise<AccessScope> {
    // Thank you, Stack Overflow :)
    return Promise.resolve({ ...values }).then(
        (result) => new Promise((resolve) => setTimeout(() => resolve(result), delay))
    );
    /*
    return Promise.reject(new Error('error message')).catch(
        (error) => new Promise((_resolve, reject) => setTimeout(() => reject(error), delay))
    );
    */
}
