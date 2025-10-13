import React from 'react';
import type { ReactNode } from 'react';
import { UserPermissionContext } from 'hooks/usePermissions';

import { fetchUserRolePermissions } from 'services/RolesService';
import useRestQuery from 'hooks/useRestQuery';

export function UserPermissionProvider({ children }: { children: ReactNode }) {
    const { data, isLoading } = useRestQuery(fetchUserRolePermissions);
    const userRolePermissions = data?.response || { resourceToAccess: {} };
    const isLoadingPermissions = isLoading;

    return (
        <UserPermissionContext.Provider value={{ userRolePermissions, isLoadingPermissions }}>
            {children}
        </UserPermissionContext.Provider>
    );
}
