import React from 'react';

import { Flex, FlexItem } from '@patternfly/react-core';

import { ClusterHealthStatus } from './clusterTypes';
import AdmissionControlPanel from './Components/AdmissionControl/AdmissionControlPanel';
import CollectorPanel from './Components/Collector/CollectorPanel';
import ScannerPanel from './Components/Scanner/ScannerPanel';
import SensorPanel from './Components/SensorPanel';

import './ClustersStatusGrid.css';

type ClustersStatusGridProps = {
    healthStatus: ClusterHealthStatus;
};

export function ClustersStatusGrid({ healthStatus }: ClustersStatusGridProps) {
    // const { scannerHealthInfo } = healthStatus;

    return (
        <Flex
            flexWrap={{ default: 'wrap' }}
            alignItems={{ default: 'alignItemsStretch' }}
            columnGap={{ default: 'columnGapMd' }}
        >
            <FlexItem className="cluster-status-panel">
                <SensorPanel healthStatus={healthStatus} />
            </FlexItem>
            <FlexItem className="cluster-status-panel">
                <CollectorPanel healthStatus={healthStatus} />
            </FlexItem>
            <FlexItem className="cluster-status-panel">
                <ScannerPanel healthStatus={healthStatus} />
            </FlexItem>
            <FlexItem className="cluster-status-panel">
                <AdmissionControlPanel healthStatus={healthStatus} />
            </FlexItem>
        </Flex>
    );
}

export default ClustersStatusGrid;
