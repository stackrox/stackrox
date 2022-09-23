import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { getHasReadPermission, getHasReadWritePermission } from 'reducers/roles';
import { IntegrationSource } from '../utils/integrationUtils';

type UseIntegrationPermissionsResponse = Record<
    IntegrationSource,
    { read: boolean; write: boolean }
>;

const authProviderState = createStructuredSelector({
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
    };
};

export default useIntegrationPermissions;
