import React, { ReactElement } from 'react';

import ClusterHealth from './ClusterHealth';

import {
    Cluster,
    clusterStatusHealthyKey,
    clusterStatusLabelMap,
    clusterStatusStyleMap,
    getCollectorStatusCountMap,
    problemText,
} from '../utils/clusters';
import { nbsp } from '../utils/health';

const healthySubtext = 'All expected collector pods are ready';

const healthyText = {
    plural: `clusters with healthy${nbsp}collectors`,
    singular: `cluster with healthy${nbsp}collectors`,
};

type Props = {
    clusters: Cluster[];
};

const CollectorStatus = ({ clusters }: Props): ReactElement => (
    <ClusterHealth
        countMap={getCollectorStatusCountMap(clusters)}
        healthyKey={clusterStatusHealthyKey}
        healthySubtext={healthySubtext}
        healthyText={healthyText}
        labelMap={clusterStatusLabelMap}
        problemText={problemText}
        styleMap={clusterStatusStyleMap}
    />
);

export default CollectorStatus;
