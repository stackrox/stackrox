import React, { ReactElement } from 'react';
import { Alert, Button, Flex, FlexItem, Switch, Text, Title } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';
import { CheckCircle } from 'react-feather';
import { ClipLoader } from 'react-spinners';

import { ClusterManagerType } from 'types/cluster.proto';
import useAnalytics, { LEGACY_CLUSTER_DOWNLOAD_YAML } from 'hooks/useAnalytics';

export type ClusterDeploymentProps = {
    clusterCheckedIn: boolean;
    createUpgraderSA: boolean;
    editing: boolean;
    isDownloadingBundle: boolean;
    managerType: ClusterManagerType;
    onFileDownload: () => void;
    toggleSA: () => void;
};

function ClusterDeployment({
    onFileDownload,
    isDownloadingBundle,
    clusterCheckedIn,
    editing,
    createUpgraderSA,
    toggleSA,
    managerType,
}: ClusterDeploymentProps): ReactElement {
    const { analyticsTrack } = useAnalytics();

    let managerTypeTitle = 'Dynamic configurations are automatically applied';
    let managerTypeText =
        'If you edited static configurations or you need to redeploy, download a new bundle.';
    if (managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR') {
        managerTypeTitle = 'Cluster labels have been saved';
        managerTypeText = 'All other cluster settings are managed by the Kubernetes operator.';
    }
    if (managerType === 'MANAGER_TYPE_HELM_CHART') {
        managerTypeTitle = 'Cluster labels have been saved';
        managerTypeText = 'All other cluster settings are managed by the Helm chart.';
    }
    // Without FlexItem element, Button stretches to column width.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            {editing && clusterCheckedIn && (
                <Alert variant="info" isInline title={managerTypeTitle} component="p">
                    {managerTypeText}
                </Alert>
            )}
            {managerType !== 'MANAGER_TYPE_KUBERNETES_OPERATOR' && (
                <>
                    <Title headingLevel="h2">Download manifest bundle</Title>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        <Title headingLevel="h3">1. Download files</Title>
                        <Text>Download the required configuration files, keys, and scripts.</Text>
                        <Switch
                            label="Cluster is configured to allow future automatic upgrades"
                            labelOff="Click to configure cluster to allow future automatic upgrades"
                            onChange={toggleSA}
                            isChecked={createUpgraderSA}
                        />
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsSm' }}
                        >
                            <FlexItem>
                                <Button
                                    variant="secondary"
                                    icon={<DownloadIcon />}
                                    onClick={() => {
                                        onFileDownload();
                                        analyticsTrack(LEGACY_CLUSTER_DOWNLOAD_YAML);
                                    }}
                                    isDisabled={isDownloadingBundle}
                                    isLoading={isDownloadingBundle}
                                >
                                    Download YAML file and keys
                                </Button>
                            </FlexItem>
                            <Text>Modify the YAML files to suit your environment if needed.</Text>
                            <Text>Do not reuse this bundle for more than one cluster.</Text>
                        </Flex>
                    </Flex>
                    <Flex direction={{ default: 'column' }}>
                        <Title headingLevel="h3">2. Deploy</Title>
                        <Text>Use the deploy script inside the bundle to set up your cluster.</Text>
                    </Flex>
                </>
            )}
            {(!editing || !clusterCheckedIn) && (
                <div className="flex flex-col text-primary-500 p-4">
                    {clusterCheckedIn ? (
                        <div className="flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
                            <div className="flex-1 text-center">
                                <CheckCircle />
                            </div>
                            <div className="flex-3 pl-2">
                                Success! The cluster has been recognized.
                            </div>
                        </div>
                    ) : (
                        <div className="flex text-primary-600 bg-primary-200 border border-solid border-primary-400 p-4 items-center">
                            <div className="text-center px-4">
                                <ClipLoader color="currentColor" loading size={20} />
                            </div>
                            <div className="flex-3 pl-2">
                                Waiting for the cluster to check in successfully...
                            </div>
                        </div>
                    )}
                </div>
            )}
        </Flex>
    );
}

export default ClusterDeployment;
