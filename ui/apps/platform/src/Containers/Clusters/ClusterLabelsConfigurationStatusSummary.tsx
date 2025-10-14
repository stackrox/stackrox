import React from 'react';
import type { ReactElement } from 'react';
import { Alert, Flex, FlexItem, Title } from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import type { Cluster, ClusterManagerType, CompleteClusterConfig } from 'types/cluster.proto';
import type { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import ClusterLabelsTable from './ClusterLabelsTable';
import ClusterStatusGrid from './ClusterStatusGrid';
import ClusterSummaryGrid from './ClusterSummaryGrid';
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
    const isManagerTypeNonConfigurable =
        managerType === 'MANAGER_TYPE_KUBERNETES_OPERATOR' ||
        managerType === 'MANAGER_TYPE_HELM_CHART';

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsLg' }}>
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {!selectedCluster.id && (
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
            {selectedCluster.id && selectedCluster.healthStatus && (
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
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
