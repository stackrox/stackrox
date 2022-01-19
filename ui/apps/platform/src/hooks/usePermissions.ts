import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { Access } from 'types/role.proto';
import { ResourceName } from 'types/roleResources';

type UsePermissionsResponse = {
    hasNoAccess: (resourceName: ResourceName) => boolean;
    hasReadAccess: (resourceName: ResourceName) => boolean;
    hasReadWriteAccess: (resourceName: ResourceName) => boolean;
};
type UserRolePermissions = (state) => { resourceToAccess: Record<ResourceName, Access> };

const stateSelector = createStructuredSelector({
    userRolePermissions: selectors.getUserRolePermissions as UserRolePermissions,
});

const usePermissions = (): UsePermissionsResponse => {
    const { userRolePermissions } = useSelector(stateSelector);

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

    return { hasNoAccess, hasReadAccess, hasReadWriteAccess };
};

export default usePermissions;
