import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { getHasReadPermission, getHasReadWritePermission } from 'reducers/roles';
import { PermissionsMap } from 'services/RolesService';
import { IntegrationSource } from '../utils/integrationUtils';

type UseIntegrationPermissionsResponse = Record<
    IntegrationSource,
    { read: boolean; write: boolean }
>;

type AuthProviderState = {
    userRolePermissions: { resourceToAccess: PermissionsMap };
};

const authProviderState = createStructuredSelector<AuthProviderState>({
    userRolePermissions: selectors.getUserRolePermissions,
});

const useIntegrationPermissions = (): UseIntegrationPermissionsResponse => {
    const { userRolePermissions } = useSelector(authProviderState);

    return {
        authProviders: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
        notifiers: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
        imageIntegrations: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
        backups: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
        signatureIntegrations: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
        cloudSources: {
            write: getHasReadWritePermission('Integration', userRolePermissions),
            read: getHasReadPermission('Integration', userRolePermissions),
        },
    };
};

export default useIntegrationPermissions;
