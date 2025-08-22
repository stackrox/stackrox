import React, { ReactElement, useState } from 'react';
import { Alert, Bullseye, Button, Divider, PageSection, Spinner } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import useAnalytics, { CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED } from 'hooks/useAnalytics';
import useRestQuery from 'hooks/useRestQuery';
import {
    ClusterRegistrationSecret,
    fetchClusterRegistrationSecrets,
} from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersClusterRegistrationSecretsPath } from 'routePaths';

import ClusterRegistrationSecretsHeader, {
    titleClusterRegistrationSecrets,
} from './ClusterRegistrationSecretsHeader';
import ClusterRegistrationSecretsTable from './ClusterRegistrationSecretsTable';
import RevokeClusterRegistrationSecretModal from './RevokeClusterRegistrationSecretModal';
import ClusterRegistrationSecretTechPreviewAlert from './ClusterRegistrationSecretTechPreviewAlert';

export type ClusterRegistrationSecretsPageProps = {
    hasWriteAccessForClusterRegistrationSecrets: boolean;
};

function ClusterRegistrationSecretsPage({
    hasWriteAccessForClusterRegistrationSecrets,
}: ClusterRegistrationSecretsPageProps): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const [clusterRegistrationSecretToRevoke, setClusterRegistrationSecretToRevoke] =
        useState<ClusterRegistrationSecret | null>(null);
    const headerActions = hasWriteAccessForClusterRegistrationSecrets ? (
        <Button
            variant="primary"
            component={LinkShim}
            href={`${clustersClusterRegistrationSecretsPath}?action=create`}
            onClick={() => {
                analyticsTrack({
                    event: CREATE_CLUSTER_REGISTRATION_SECRET_CLICKED,
                    properties: { source: 'Cluster Registration Secrets' },
                });
            }}
        >
            Create cluster registration secret
        </Button>
    ) : null;

    const {
        data: dataForFetch,
        error: errorForFetch,
        isLoading: isFetching,
        refetch,
    } = useRestQuery(fetchClusterRegistrationSecrets);

    function onCloseModal(wasRevoked: boolean) {
        setClusterRegistrationSecretToRevoke(null);
        if (wasRevoked) {
            refetch();
        }
    }

    return (
        <>
            <ClusterRegistrationSecretsHeader
                headerActions={headerActions}
                title={titleClusterRegistrationSecrets}
            />

            <Divider component="div" />
            <PageSection component="div" variant="light">
                <ClusterRegistrationSecretTechPreviewAlert />
            </PageSection>
            <PageSection component="div">
                {isFetching ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : errorForFetch ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch cluster cluster registration secrets"
                        component="p"
                        isInline
                    >
                        {getAxiosErrorMessage(errorForFetch)}
                    </Alert>
                ) : (
                    <>
                        <ClusterRegistrationSecretsTable
                            hasWriteAccessForClusterRegistrationSecrets={
                                hasWriteAccessForClusterRegistrationSecrets
                            }
                            clusterRegistrationSecrets={dataForFetch?.items ?? []}
                            setClusterRegistrationSecretToRevoke={
                                setClusterRegistrationSecretToRevoke
                            }
                        />
                        {clusterRegistrationSecretToRevoke && (
                            <RevokeClusterRegistrationSecretModal
                                clusterRegistrationSecret={clusterRegistrationSecretToRevoke}
                                onCloseModal={onCloseModal}
                            />
                        )}
                    </>
                )}
            </PageSection>
        </>
    );
}

export default ClusterRegistrationSecretsPage;
