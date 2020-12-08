import React, { ReactElement } from 'react';

import ClusterHealth from './ClusterHealth';

import {
    Cluster,
    clusterStatusHealthyKey,
    clusterStatusLabelMap,
    clusterStatusStyleMap,
    getSensorStatusCountMap,
    problemText,
} from '../utils/clusters';
import { nbsp } from '../utils/health';

const healthySubtext = 'All sensors last contacted less than 1 minute ago';

const healthyText = {
    plural: `clusters with healthy${nbsp}sensors`,
    singular: `cluster with healthy${nbsp}sensor`,
};

type Props = {
    clusters: Cluster[];
};

const SensorStatus = ({ clusters }: Props): ReactElement => (
    <ClusterHealth
        countMap={getSensorStatusCountMap(clusters)}
        healthyKey={clusterStatusHealthyKey}
        healthySubtext={healthySubtext}
        healthyText={healthyText}
        labelMap={clusterStatusLabelMap}
        problemText={problemText}
        styleMap={clusterStatusStyleMap}
    />
);

export default SensorStatus;
