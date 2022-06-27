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
            write: getHasReadWritePermission('APIToken', userRolePermissions),
            read: getHasReadPermission('APIToken', userRolePermissions),
        },
        notifiers: {
            write: getHasReadWritePermission('Notifier', userRolePermissions),
            read: getHasReadPermission('Notifier', userRolePermissions),
        },
        imageIntegrations: {
            write: getHasReadWritePermission('ImageIntegration', userRolePermissions),
            read: getHasReadPermission('ImageIntegration', userRolePermissions),
        },
        backups: {
            write: getHasReadWritePermission('BackupPlugins', userRolePermissions),
            read: getHasReadPermission('BackupPlugins', userRolePermissions),
        },
        signatureIntegrations: {
            write: getHasReadWritePermission('SignatureIntegration', userRolePermissions),
            read: getHasReadPermission('SignatureIntegration', userRolePermissions),
        },
    };
};

export default useIntegrationPermissions;
