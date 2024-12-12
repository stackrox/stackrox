import React, { ReactElement, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Alert, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { fetchClusterRegistrationSecrets } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterRegistrationSecretDescription from './ClusterRegistrationSecretDescription';
import ClusterRegistrationSecretsHeader from './ClusterRegistrationSecretsHeader';
import RevokeClusterRegistrationSecretModal from './RevokeClusterRegistrationSecretModal';

export type ClusterRegistrationSecretPageProps = {
    hasWriteAccessForClusterRegistrationSecrets: boolean;
    id: string;
};

function ClusterRegistrationSecretPage({ hasWriteAccessForClusterRegistrationSecrets, id }: ClusterRegistrationSecretPageProps): ReactElement {
    const history = useHistory();
    const [isRevoking, setIsRevoking] = useState(false);

    const {
        data: dataForFetch,
        isLoading: isFetching,
        error: errorForFetch,
    } = useRestQuery(fetchClusterRegistrationSecrets);

    const clusterRegistrationSecret = dataForFetch?.response?.items.find(
        (clusterRegistrationSecretArg) => clusterRegistrationSecretArg.id === id
    );

    function onClickRevoke() {
        setIsRevoking(true);
    }

    function onCloseModal(wasRevoked: boolean) {
        setIsRevoking(false);
        if (wasRevoked) {
            history.goBack(); // to table
        }
    }

    const headerActions =
        hasWriteAccessForClusterRegistrationSecrets && clusterRegistrationSecret ? (
            <Button
                variant="danger"
                isDisabled={isRevoking}
                isLoading={isRevoking}
                onClick={onClickRevoke}
            >
                Revoke cluster registration secret
            </Button>
        ) : null;

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <ClusterRegistrationSecretsHeader headerActions={headerActions} title="Cluster registration secret" />
            <PageSection component="div">
                {isFetching ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : errorForFetch ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster registration secrets"
                        component="p"
                        isInline
                    >
                        {getAxiosErrorMessage(errorForFetch)}
                    </Alert>
                ) : clusterRegistrationSecret ? (
                    <>
                        <ClusterRegistrationSecretDescription clusterRegistrationSecret={clusterRegistrationSecret} />
                        {isRevoking && (
                            <RevokeClusterRegistrationSecretModal
                            clusterRegistrationSecret={clusterRegistrationSecret}
                                onCloseModal={onCloseModal}
                            />
                        )}
                    </>
                ) : (
                    <Alert
                        variant="warning"
                        title="Unable to find cluster registration secret"
                        component="p"
                        isInline
                    />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default ClusterRegistrationSecretPage;
