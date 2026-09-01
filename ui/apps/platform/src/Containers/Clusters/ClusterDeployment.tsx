import type { ReactElement } from 'react';
import {
    Alert,
    Button,
    Content,
    Flex,
    FlexItem,
    Spinner,
    Switch,
    Title,
} from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';

import type { ClusterManagerType } from 'types/cluster.proto';
import useAnalytics, { LEGACY_CLUSTER_DOWNLOAD_YAML } from 'hooks/useAnalytics';

import InstallMethodDeprecationAlert from './Components/InstallMethodDeprecationAlert';

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

    const isHelmOrOperatorManaged =
        managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR' ||
        managerType === 'MANAGER_TYPE_HELM_CHART';

    const showPostCheckInSavedConfigAlert = editing && clusterCheckedIn && !isHelmOrOperatorManaged;

    // Without FlexItem element, Button stretches to column width.
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            {showPostCheckInSavedConfigAlert && (
                <Alert
                    variant="info"
                    isInline
                    title="Dynamic configurations are automatically applied"
                    component="p"
                >
                    If you edited static configurations or you need to redeploy, download a new
                    bundle.
                </Alert>
            )}
            {managerType !== 'MANAGER_TYPE_KUBERNETES_OPERATOR' && (
                <Flex direction={{ default: 'column' }}>
                    <FlexItem spacer={{ default: 'spacerLg' }}>
                        <InstallMethodDeprecationAlert deprecationMessage="The legacy manifest bundle installation method is deprecated since version 4.9 and will be removed in 5.1." />
                    </FlexItem>
                    <Title headingLevel="h2">Download manifest bundle</Title>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        <Title headingLevel="h3">1. Configure possibility of future upgrades</Title>
                        <Content component="p">
                            Configuring clusters for future upgrades creates a powerful service
                            account in your secured cluster that will be used to perform the
                            upgrades. This is a prerequisite for automated or on-click upgrades of
                            legacy-installed Secured Clusters to work.
                        </Content>
                        <Switch
                            label="Configured for upgrades: Secured Clusters can be upgraded to match Central’s version."
                            onChange={toggleSA}
                            isChecked={createUpgraderSA}
                        />
                        <Title headingLevel="h3">2. Download files</Title>
                        <Content component="p">
                            Download the required configuration files, keys, and scripts.
                        </Content>
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
                            <Content component="p">
                                Modify the YAML files to suit your environment if needed.
                            </Content>
                            <Content component="p">
                                Do not reuse this bundle for more than one cluster.
                            </Content>
                        </Flex>
                    </Flex>
                    <Flex direction={{ default: 'column' }}>
                        <Title headingLevel="h3">3. Deploy</Title>
                        <Content component="p">
                            Use the deploy script inside the bundle to set up your cluster.
                        </Content>
                    </Flex>
                </Flex>
            )}
            {(!editing || !clusterCheckedIn) &&
                (clusterCheckedIn ? (
                    <Alert
                        variant="success"
                        isInline
                        title="Success! The cluster has been recognized."
                        component="p"
                    />
                ) : (
                    <Alert
                        variant="info"
                        isInline
                        title="Waiting for the cluster to check in successfully..."
                        component="p"
                        customIcon={<Spinner size="md" />}
                    />
                ))}
        </Flex>
    );
}

export default ClusterDeployment;
