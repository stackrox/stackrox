import React from 'react';
import type { ReactElement } from 'react';
import { Alert, Flex, FlexItem, Title } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { Cluster, ClusterManagerType } from 'types/cluster.proto';
import type { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import ClusterLabelsTable from './ClusterLabelsTable';
import ClusterStatusGrid from './ClusterStatusGrid';
import ClusterSummaryGrid from './ClusterSummaryGrid';
import ClusterSummaryLegacy from './Components/ClusterSummaryLegacy';
import DynamicConfigurationForm from './DynamicConfigurationForm';
import StaticConfigurationForm from './StaticConfigurationForm';

type ClusterLabelsConfigurationStatusSummaryProps = {
    centralVersion: string;
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
    selectedCluster: Cluster;
    managerType: ClusterManagerType;
    handleChange: (path: string, value: boolean | number | string) => void;
    handleChangeAdmissionControllerEnforcementBehavior: (value: boolean) => void;
    handleChangeLabels: (labels) => void;
};

function ClusterLabelsConfigurationStatusSummary({
    centralVersion,
    clusterRetentionInfo,
    selectedCluster,
    managerType,
    handleChange,
    handleChangeAdmissionControllerEnforcementBehavior,
    handleChangeLabels,
}: ClusterLabelsConfigurationStatusSummaryProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isAdmissionControllerConfigEnabled = isFeatureFlagEnabled(
        'ROX_ADMISSION_CONTROLLER_CONFIG'
    );
    const isManagerTypeNonConfigurable =
        managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR' ||
        managerType === 'MANAGER_TYPE_HELM_CHART';

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {selectedCluster.id && selectedCluster.healthStatus ? (
                isAdmissionControllerConfigEnabled ? null : (
                    <ClusterSummaryLegacy
                        healthStatus={selectedCluster.healthStatus}
                        status={selectedCluster.status}
                        centralVersion={centralVersion}
                        clusterId={selectedCluster.id}
                        autoRefreshEnabled={selectedCluster.sensorCapabilities?.includes(
                            'SecuredClusterCertificatesRefresh'
                        )}
                        clusterRetentionInfo={clusterRetentionInfo}
                        isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                    />
                )
            ) : (
                <Alert variant="warning" isInline title="Legacy installation method" component="p">
                    <Flex direction={{ default: 'column' }}>
                        <FlexItem>
                            <p>
                                To avoid extra operational complexity, use a{' '}
                                <strong>cluster init bundle</strong> with either of the following
                                installation methods:
                            </p>
                            <p>
                                <strong>Operator</strong> for Red Hat OpenShift
                            </p>
                            <p>
                                <strong>Helm chart</strong> for other platforms
                            </p>
                        </FlexItem>
                        <FlexItem>
                            <p>
                                Only use the legacy installation method if you have a specific
                                installation need that requires using this method.
                            </p>
                        </FlexItem>
                    </Flex>
                </Alert>
            )}
            {selectedCluster.id && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                    <Title headingLevel="h2">Cluster labels</Title>
                    <ClusterLabelsTable
                        labels={selectedCluster?.labels ?? {}}
                        handleChangeLabels={handleChangeLabels}
                        hasAction
                    />
                </Flex>
            )}
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel="h2">Static configuration (requires deployment)</Title>
                <StaticConfigurationForm
                    isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                    handleChange={handleChange}
                    selectedCluster={selectedCluster}
                />
            </Flex>
            <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                <Title headingLevel="h2">Dynamic configuration (syncs with Sensor)</Title>
                <DynamicConfigurationForm
                    clusterType={selectedCluster.type}
                    dynamicConfig={selectedCluster.dynamicConfig}
                    handleChange={handleChange}
                    handleChangeAdmissionControllerEnforcementBehavior={
                        handleChangeAdmissionControllerEnforcementBehavior
                    }
                    helmConfig={selectedCluster.helmConfig}
                    isManagerTypeNonConfigurable={isManagerTypeNonConfigurable}
                />
            </Flex>
            {selectedCluster.id &&
                selectedCluster.healthStatus &&
                isAdmissionControllerConfigEnabled && (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Title headingLevel="h2">Cluster status</Title>
                        <Flex
                            direction={{ default: 'column' }}
                            spaceItems={{ default: 'spaceItemsMd' }}
                        >
                            <ClusterStatusGrid healthStatus={selectedCluster.healthStatus} />
                            <ClusterSummaryGrid
                                centralVersion={centralVersion}
                                clusterInfo={selectedCluster}
                                clusterRetentionInfo={clusterRetentionInfo}
                            />
                        </Flex>
                    </Flex>
                )}
        </Flex>
    );
}

export default ClusterLabelsConfigurationStatusSummary;
