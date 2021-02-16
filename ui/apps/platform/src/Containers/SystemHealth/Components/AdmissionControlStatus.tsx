import React, { ReactElement } from 'react';

import ClusterHealth from './ClusterHealth';

import {
    Cluster,
    clusterStatusHealthyKey,
    clusterStatusLabelMap,
    clusterStatusStyleMap,
    getAdmissionControlStatusCountMap,
    problemText,
} from '../utils/clusters';
import { nbsp } from '../utils/health';

const healthySubtext = 'All expected admission control pods are ready';

const healthyText = {
    plural: `clusters with healthy${nbsp}admission control pods`,
    singular: `cluster with healthy${nbsp}admission control pods`,
};

type Props = {
    clusters: Cluster[];
};

const AdmissionControlStatus = ({ clusters }: Props): ReactElement => (
    <ClusterHealth
        countMap={getAdmissionControlStatusCountMap(clusters)}
        healthyKey={clusterStatusHealthyKey}
        healthySubtext={healthySubtext}
        healthyText={healthyText}
        labelMap={clusterStatusLabelMap}
        problemText={problemText}
        styleMap={clusterStatusStyleMap}
    />
);

export default AdmissionControlStatus;
