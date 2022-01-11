import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { Access, ResourceName } from 'types/role.proto';

type UsePermissionsResponse = {
    hasNoAccess: (resourceName: ResourceName) => boolean;
    hasReadAccess: (resourceName: ResourceName) => boolean;
    hasReadWriteAccess: (resourceName: ResourceName) => boolean;
    currentUserName: string;
};
type UserRolePermissions = (state) => { resourceToAccess: Record<ResourceName, Access> };
type CurrentUserName = (state) => string;

const stateSelector = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions as UserRolePermissions,
    // @TODO: currentUserName can be moved into it's own hook to separate concerns
    currentUserName: selectors.getCurrentUserName as CurrentUserName,
});

const usePermissions = (): UsePermissionsResponse => {
    const { userRolePermissions, currentUserName } = useSelector(stateSelector);

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

    return { hasNoAccess, hasReadAccess, hasReadWriteAccess, currentUserName };
};

export default usePermissions;
