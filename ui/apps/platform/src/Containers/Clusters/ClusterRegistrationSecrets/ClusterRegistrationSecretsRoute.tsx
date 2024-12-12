import React, { ReactElement } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import qs from 'qs';

import useAuthStatus from 'hooks/useAuthStatus'; // TODO after 4.4 release

import ClusterRegistrationSecretForm from './ClusterRegistrationSecretForm';
import ClusterRegistrationSecretPage from './ClusterRegistrationSecretPage';
import ClusterRegistrationSecretsPage from './ClusterRegistrationSecretsPage';

function hasCreateAction(search: string) {
    const { action } = qs.parse(search, { ignoreQueryPrefix: true });
    return action === 'create';
}

function ClusterRegistrationSecretsRoute(): ReactElement {
    const { currentUser } = useAuthStatus();
    const hasAdminRole = Boolean(currentUser?.userInfo?.roles.some(({ name }) => name === 'Admin')); // optional chaining just in case of the unexpected
    const hasWriteAccessForClusterRegistrationSecrets = hasAdminRole; // TODO after 4.4 release becomes redundant

    const { search } = useLocation();
    const { id } = useParams(); // see clustersClusterRegistrationSecretPathWithParam in routePaths.ts

    /*
    // TODO after 4.4 release
    if (!hasAdminRole) {
        return <NotFoundPage />; // factor out reusable component from Body.tsx file
    }
    */

    const isCreateAction = hasWriteAccessForClusterRegistrationSecrets && hasCreateAction(search);

    if (id) {
        return (
            <ClusterRegistrationSecretPage hasWriteAccessForClusterRegistrationSecrets={hasWriteAccessForClusterRegistrationSecrets} id={id} />
        );
    }

    if (hasWriteAccessForClusterRegistrationSecrets && isCreateAction) {
        return <ClusterRegistrationSecretForm />;
    }

    return <ClusterRegistrationSecretsPage hasWriteAccessForClusterRegistrationSecrets={hasWriteAccessForClusterRegistrationSecrets} />;
}

export default ClusterRegistrationSecretsRoute;
