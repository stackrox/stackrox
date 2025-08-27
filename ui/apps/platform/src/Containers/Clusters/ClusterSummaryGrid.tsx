import React from 'react';

import { Grid, GridItem } from '@patternfly/react-core';

import type { Cluster } from 'types/cluster.proto';
import type { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import ClusterDeletion from './Components/ClusterDeletion';
import ClusterHealthPanel from './Components/ClusterHealthPanel';
import ClusterMetadata from './Components/ClusterMetadata';
import CredentialExpiration from './Components/CredentialExpiration';
import SensorUpgradePanel from './Components/SensorUpgradePanel';
import type { CertExpiryStatus } from './clusterTypes';

type ClusterSummaryGridProps = {
    centralVersion: string;
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
    clusterInfo: Cluster;
};

export function ClusterSummaryGrid({
    centralVersion,
    clusterRetentionInfo,
    clusterInfo,
}: ClusterSummaryGridProps) {
    return (
        <Grid hasGutter>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <ClusterHealthPanel header="Cluster metadata">
                    <ClusterMetadata status={clusterInfo.status} />
                </ClusterHealthPanel>
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <SensorUpgradePanel
                    centralVersion={centralVersion}
                    sensorVersion={clusterInfo.status?.sensorVersion}
                    upgradeStatus={clusterInfo.status?.upgradeStatus}
                    actionProps={{
                        clusterId: clusterInfo.id,
                        upgradeSingleCluster: () => {},
                    }}
                />
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <ClusterHealthPanel header="Credential expiration">
                    <CredentialExpiration
                        certExpiryStatus={clusterInfo.status?.certExpiryStatus as CertExpiryStatus}
                        autoRefreshEnabled={clusterInfo.sensorCapabilities?.includes(
                            'SecuredClusterCertificatesRefresh'
                        )}
                        isList
                    />
                </ClusterHealthPanel>
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <ClusterHealthPanel header="Cluster deletion">
                    <ClusterDeletion clusterRetentionInfo={clusterRetentionInfo} />
                </ClusterHealthPanel>
            </GridItem>
        </Grid>
    );
}

export default ClusterSummaryGrid;
