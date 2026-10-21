import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Flex,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useMetadata from 'hooks/useMetadata';
import { fetchClusterWithRetentionInformation } from 'services/ClustersService';
import type { Cluster, ClusterManagerType } from 'types/cluster.proto';
import type { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersBasePath } from 'routePaths';

import ClusterLabelsConfigurationStatusSummary from './ClusterLabelsConfigurationStatusSummary';

type MessageState = {
    variant: 'info' | 'danger';
    title: string;
    text: string;
};

export type ClusterPageProps = {
    clusterId: string;
};

function ClusterPage({ clusterId }: ClusterPageProps): ReactElement {
    const metadata = useMetadata();

    const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null);
    const [clusterRetentionInfo, setClusterRetentionInfo] =
        useState<DecommissionedClusterRetentionInfo>(null);
    const [messageState, setMessageState] = useState<MessageState | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    function managerType(cluster: Partial<Cluster> | null): ClusterManagerType {
        return cluster?.helmConfig && cluster.managedBy === 'MANAGER_TYPE_UNKNOWN'
            ? 'MANAGER_TYPE_HELM_CHART'
            : (cluster?.managedBy ?? 'MANAGER_TYPE_UNKNOWN');
    }

    useEffect(() => {
        if (clusterId) {
            setMessageState(null);
            setIsLoading(true);
            fetchClusterWithRetentionInformation(clusterId)
                .then((clusterResponse) => {
                    const { cluster } = clusterResponse;
                    setSelectedCluster(cluster);
                    setClusterRetentionInfo(clusterResponse.clusterRetentionInfo);

                    switch (managerType(cluster)) {
                        case 'MANAGER_TYPE_HELM_CHART':
                            setMessageState({
                                variant: 'info',
                                title: 'Helm-managed cluster',
                                text: 'This is a Helm-managed cluster. The settings of Helm-managed clusters cannot be changed here, please ask your DevOps team to change the settings by updating the Helm values.',
                            });
                            break;
                        case 'MANAGER_TYPE_KUBERNETES_OPERATOR':
                            setMessageState({
                                variant: 'info',
                                title: 'Operator-managed cluster',
                                text: 'This is an operator-managed cluster. The settings of operator-managed clusters cannot be changed here and must instead be changed by updating its SecuredCluster custom resource (CR).',
                            });
                            break;
                        default:
                            break;
                    }
                })
                .catch((error) => {
                    setMessageState({
                        variant: 'danger',
                        title: 'error retrieving the configuration for the cluster',
                        text: getAxiosErrorMessage(error),
                    });
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [clusterId]);

    const selectedClusterName = selectedCluster?.name ?? '';

    return (
        <>
            <PageTitle title="Cluster" />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{selectedClusterName}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Title headingLevel="h1">{selectedClusterName}</Title>
                    {!!messageState && (
                        <Alert
                            variant={messageState.variant}
                            isInline
                            title={messageState.title}
                            component="p"
                        >
                            {messageState.text}
                        </Alert>
                    )}
                    {isLoading && (
                        <Bullseye>
                            <Spinner />
                        </Bullseye>
                    )}
                    {selectedCluster && (
                        <ClusterLabelsConfigurationStatusSummary
                            centralVersion={metadata.version}
                            clusterRetentionInfo={clusterRetentionInfo}
                            selectedCluster={selectedCluster}
                            managerType={managerType(selectedCluster)}
                        />
                    )}
                </Flex>
            </PageSection>
        </>
    );
}

export default ClusterPage;
