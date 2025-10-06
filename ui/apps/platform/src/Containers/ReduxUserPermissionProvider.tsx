import React from 'react';
import type { ReactNode } from 'react';
import { useSelector } from 'react-redux';
import type { Access } from 'types/role.proto';
import type { ResourceName } from 'types/roleResources';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { UserPermissionContext } from 'hooks/usePermissions';

const stateSelector = createStructuredSelector<{
    userRolePermissions: { resourceToAccess: Partial<Record<ResourceName, Access>> } | null;
    isLoadingPermissions: boolean;
}>({
    userRolePermissions: selectors.getUserRolePermissions,
    isLoadingPermissions: selectors.getIsLoadingUserRolePermissions,
});

export default function ReduxUserPermissionProvider({ children }: { children: ReactNode }) {
    const { userRolePermissions, isLoadingPermissions } = useSelector(stateSelector);

    return (
        <UserPermissionContext.Provider value={{ userRolePermissions, isLoadingPermissions }}>
            {children}
        </UserPermissionContext.Provider>
    );
}
