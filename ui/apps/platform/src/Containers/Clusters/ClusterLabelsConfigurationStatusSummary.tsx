import type { ReactElement } from 'react';
import { Alert, Content, Flex, Title } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import type { Cluster, ClusterManagerType, CompleteClusterConfig } from 'types/cluster.proto';
import type { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';
import ClusterStatusGrid from './ClusterStatusGrid';
import ClusterSummaryGrid from './ClusterSummaryGrid';
import SensorCompatibilityPanel from './Components/SensorCompatibilityPanel';
import DynamicConfigurationForm from './DynamicConfigurationForm';
import StaticConfigurationForm from './StaticConfigurationForm';
// Delete whenever deprecated properties are deleted.
function getClusterHasDefaultsForAdmissionController(helmConfig: CompleteClusterConfig) {
    return (
        helmConfig.staticConfig.admissionController &&
        helmConfig.staticConfig.admissionControllerUpdates &&
        helmConfig.staticConfig.admissionControllerEvents &&
        helmConfig.dynamicConfig?.admissionControllerConfig?.scanInline &&
        helmConfig.dynamicConfig?.admissionControllerConfig?.enabled &&
        helmConfig.dynamicConfig?.admissionControllerConfig?.enforceOnUpdates
    );
}

type ClusterLabelsConfigurationStatusSummaryProps = {
    centralVersion: string;
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
    selectedCluster: Cluster;
    managerType: ClusterManagerType;
};

function ClusterLabelsConfigurationStatusSummary({
    centralVersion,
    clusterRetentionInfo,
    selectedCluster,
    managerType,
}: ClusterLabelsConfigurationStatusSummaryProps): ReactElement {
    const metadata = useMetadata();
    const compatibleVersions = metadata?.compatibleSensorVersions ?? [];

    const isManagerTypeNonConfigurable =
        managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR' ||
        managerType === 'MANAGER_TYPE_HELM_CHART';

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            {selectedCluster.id && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Title headingLevel="h2">Cluster labels</Title>
                    {Object.keys(selectedCluster?.labels ?? {}).length === 0 ? (
                        <Content component="p">No labels</Content>
                    ) : (
                        <Table variant="compact" aria-label="Cluster labels">
                            <Thead>
                                <Tr>
                                    <Th>Key</Th>
                                    <Th>Value</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {Object.entries(selectedCluster.labels).map(([key, value]) => (
                                    <Tr key={key}>
                                        <Td dataLabel="Key" modifier="breakWord">
                                            {key}
                                        </Td>
                                        <Td dataLabel="Value" modifier="breakWord">
                                            {value}
                                        </Td>
                                    </Tr>
                                ))}
                            </Tbody>
                        </Table>
                    )}
                </Flex>
            )}
            {isManagerTypeNonConfigurable &&
                selectedCluster.helmConfig &&
                !getClusterHasDefaultsForAdmissionController(selectedCluster.helmConfig) && (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Alert
                            variant="warning"
                            isInline
                            title="Admission controller configuration of this secured cluster differs from default configuration"
                            component="p"
                        >
                            For more information, see{' '}
                            <ExternalLink>
                                <a
                                    href="https://access.redhat.com/solutions/7130669"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    Knowledge Centered Support solution
                                </a>
                            </ExternalLink>
                        </Alert>
                    </Flex>
                )}
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel="h2">Static configuration (requires deployment)</Title>
                <StaticConfigurationForm
                    isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                    selectedCluster={selectedCluster}
                    dynamicConfig={selectedCluster.dynamicConfig}
                    helmConfig={selectedCluster.helmConfig}
                />
            </Flex>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel="h2">Dynamic configuration (syncs with Sensor)</Title>
                <DynamicConfigurationForm
                    dynamicConfig={selectedCluster.dynamicConfig}
                    helmConfig={selectedCluster.helmConfig}
                />
            </Flex>
            {selectedCluster.id && selectedCluster.healthStatus && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Title headingLevel="h2">Cluster status</Title>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        <ClusterStatusGrid healthStatus={selectedCluster.healthStatus} />
                        <ClusterSummaryGrid
                            clusterInfo={selectedCluster}
                            clusterRetentionInfo={clusterRetentionInfo}
                        />
                    </Flex>
                </Flex>
            )}
            {selectedCluster.id && selectedCluster.status && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Title headingLevel="h2">Compatibility status</Title>
                    <SensorCompatibilityPanel
                        compatibility={selectedCluster.status.sensorVersionCompatibility}
                        compatibleVersions={compatibleVersions}
                        sensorVersion={selectedCluster.status.sensorVersion}
                        centralVersion={centralVersion}
                    />
                </Flex>
            )}
        </Flex>
    );
}

export default ClusterLabelsConfigurationStatusSummary;
