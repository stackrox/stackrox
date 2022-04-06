import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { Access } from 'types/role.proto';
import { ResourceName } from 'types/roleResources';

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
        return access === 'NO_ACCESS';
    }

    function hasReadAccess(resourceName: ResourceName) {
        const access = userRolePermissions?.resourceToAccess[resourceName];
        return access === 'READ_ACCESS' || access === 'READ_WRITE_ACCESS';
    }

    function hasReadWriteAccess(resourceName: ResourceName) {
        const access = userRolePermissions?.resourceToAccess[resourceName];
        return access === 'READ_WRITE_ACCESS';
    }

    return { hasNoAccess, hasReadAccess, hasReadWriteAccess, isLoadingPermissions };
};

export default usePermissions;
