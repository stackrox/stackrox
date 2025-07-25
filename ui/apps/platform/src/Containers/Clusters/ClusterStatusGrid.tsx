import React from 'react';

import { Grid, GridItem } from '@patternfly/react-core';

import { ClusterHealthStatus } from 'types/cluster.proto';

import AdmissionControlPanel from './Components/AdmissionControl/AdmissionControlPanel';
import CollectorPanel from './Components/Collector/CollectorPanel';
import ScannerPanel from './Components/Scanner/ScannerPanel';
import SensorPanel from './Components/SensorPanel';

type ClusterStatusGridProps = {
    healthStatus: ClusterHealthStatus;
};

export function ClusterStatusGrid({ healthStatus }: ClusterStatusGridProps) {
    return (
        <Grid hasGutter>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <SensorPanel healthStatus={healthStatus} />
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <CollectorPanel healthStatus={healthStatus} />
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <AdmissionControlPanel healthStatus={healthStatus} />
            </GridItem>
            <GridItem span={12} lg={6} xl={3} className="cluster-status-panel">
                <ScannerPanel healthStatus={healthStatus} />
            </GridItem>
        </Grid>
    );
}

export default ClusterStatusGrid;
