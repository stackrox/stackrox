import { defaultMinimalReadAccessResources } from 'constants/accessControl';
import type { AccessLevel, PermissionsMap, PermissionSet } from 'services/RolesService';

/*
 * Return a new permission set with default minimal read access.
 */
export function getNewPermissionSet(resources: string[]): PermissionSet {
    const resourceToAccess: PermissionsMap = {};
    resources.forEach((resource) => {
        resourceToAccess[resource] = defaultMinimalReadAccessResources.includes(resource)
            ? 'READ_ACCESS'
            : 'NO_ACCESS';
    });

    return {
        id: '',
        name: '',
        description: '',
        resourceToAccess,
    };
}

function getCompletePermissions(permissions: PermissionsMap, resources: string[]): PermissionsMap {
    const completePermissions: PermissionsMap = {};

    resources.forEach((resource) => {
        completePermissions[resource] = permissions[resource] ?? 'NO_ACCESS';
    });

    return completePermissions;
}

/*
 * Make sure permission set has all resources in PermissionsTable rendered by PermissionSetForm.
 * Needed for the following default permission sets:
 * Continuous Integration
 * Sensor Creator
 * None
 * Also in case new resources are added.
 */
export function getCompletePermissionSet(
    permissionSet: PermissionSet,
    resources: string[]
): PermissionSet {
    return {
        ...permissionSet,
        resourceToAccess: getCompletePermissions(permissionSet.resourceToAccess, resources),
    };
}

/*
 * Return whether access level is (at least) read access.
 */
export function getIsReadAccess(accessLevel: AccessLevel): boolean {
    return accessLevel === 'READ_ACCESS' || accessLevel === 'READ_WRITE_ACCESS';
}

/*
 * Return whether access level is write access.
 */
export function getIsWriteAccess(accessLevel: AccessLevel): boolean {
    return accessLevel === 'READ_WRITE_ACCESS';
}

/*
 * Return count of resources which have (at least) read access.
 */
export function getReadAccessCount(resourceToAccess: PermissionsMap): number {
    let count = 0;

    Object.values(resourceToAccess).forEach((accessLevel) => {
        if (getIsReadAccess(accessLevel)) {
            count += 1;
        }
    });

    return count;
}

/*
 * Return count of resources which have write access.
 */
export function getWriteAccessCount(resourceToAccess: PermissionsMap): number {
    let count = 0;

    Object.values(resourceToAccess).forEach((accessLevel) => {
        if (getIsWriteAccess(accessLevel)) {
            count += 1;
        }
    });

    return count;
}
