import { createContext, useContext } from 'react';

import type { Access } from 'types/role.proto';
import type { ResourceName } from 'types/roleResources';
import { replacedResourceMapping } from 'constants/accessControl';

export type HasNoAccess = (resourceName: ResourceName) => boolean;
export type HasReadAccess = (resourceName: ResourceName) => boolean;
export type HasReadWriteAccess = (resourceName: ResourceName) => boolean;

export const UserPermissionContext = createContext<{
    userRolePermissions: { resourceToAccess: Partial<Record<ResourceName, Access>> } | null;
    isLoadingPermissions: boolean;
}>({
    userRolePermissions: { resourceToAccess: {} },
    isLoadingPermissions: false,
});

type UsePermissionsResponse = {
    hasNoAccess: HasNoAccess;
    hasReadAccess: HasReadAccess;
    hasReadWriteAccess: HasReadWriteAccess;
    isLoadingPermissions: boolean;
};

const usePermissions = (): UsePermissionsResponse => {
    const { userRolePermissions, isLoadingPermissions } = useContext(UserPermissionContext);

    function hasNoAccess(resourceName: ResourceName) {
        const access = userRolePermissions?.resourceToAccess[resourceName];
        if (access === 'NO_ACCESS') {
            return true;
        }

        if (replacedResourceMapping.has(resourceName)) {
            const replacedResourceAccess =
                userRolePermissions?.resourceToAccess[
                    replacedResourceMapping.get(resourceName) as ResourceName
                ];
            return replacedResourceAccess === 'NO_ACCESS';
        }
        return false;
    }

    function hasReadAccess(resourceName: ResourceName) {
        const access = userRolePermissions?.resourceToAccess[resourceName];
        if (access === 'READ_ACCESS' || access === 'READ_WRITE_ACCESS') {
            return true;
        }

        if (replacedResourceMapping.has(resourceName)) {
            const replacedResourceAccess =
                userRolePermissions?.resourceToAccess[
                    replacedResourceMapping.get(resourceName) as ResourceName
                ];
            return (
                replacedResourceAccess === 'READ_ACCESS' ||
                replacedResourceAccess === 'READ_WRITE_ACCESS'
            );
        }
        return false;
    }

    function hasReadWriteAccess(resourceName: ResourceName) {
        const access = userRolePermissions?.resourceToAccess[resourceName];
        if (access === 'READ_WRITE_ACCESS') {
            return true;
        }

        if (replacedResourceMapping.has(resourceName)) {
            const replacedResourceAccess =
                userRolePermissions?.resourceToAccess[
                    replacedResourceMapping.get(resourceName) as ResourceName
                ];
            return replacedResourceAccess === 'READ_WRITE_ACCESS';
        }
        return false;
    }

    return { hasNoAccess, hasReadAccess, hasReadWriteAccess, isLoadingPermissions };
};

export default usePermissions;
