import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { Access } from 'types/role.proto';
import { ResourceName } from 'types/roleResources';
import { replacedResourceMapping } from 'constants/accessControl';

export type HasNoAccess = (resourceName: ResourceName) => boolean;
export type HasReadAccess = (resourceName: ResourceName) => boolean;
export type HasReadWriteAccess = (resourceName: ResourceName) => boolean;

type UsePermissionsResponse = {
    hasNoAccess: HasNoAccess;
    hasReadAccess: HasReadAccess;
    hasReadWriteAccess: HasReadWriteAccess;
    isLoadingPermissions: boolean;
};

const stateSelector = createStructuredSelector<{
    userRolePermissions: { resourceToAccess: Record<ResourceName, Access> };
    isLoadingPermissions: boolean;
}>({
    userRolePermissions: selectors.getUserRolePermissions,
    isLoadingPermissions: selectors.getIsLoadingUserRolePermissions,
});

const usePermissions = (): UsePermissionsResponse => {
    const { userRolePermissions, isLoadingPermissions } = useSelector(stateSelector);

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
