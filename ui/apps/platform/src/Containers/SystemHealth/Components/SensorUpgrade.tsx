import React, { ReactElement } from 'react';

import ClusterHealth from './ClusterHealth';

import {
    Cluster,
    getSensorUpgradeCountMap,
    problemText,
    sensorUpgradeHealthyKey,
    sensorUpgradeLabelMap,
    sensorUpgradeStyleMap,
} from '../utils/clusters';
import { nbsp } from '../utils/health';

const healthySubtext = 'All sensor versions match central version';

const healthyText = {
    plural: `clusters up${nbsp}to${nbsp}date with${nbsp}central`,
    singular: `cluster up${nbsp}to${nbsp}date with${nbsp}central`,
};

type Props = {
    clusters: Cluster[];
};

const SensorUpgrade = ({ clusters }: Props): ReactElement => (
    <ClusterHealth
        countMap={getSensorUpgradeCountMap(clusters)}
        healthyKey={sensorUpgradeHealthyKey}
        healthySubtext={healthySubtext}
        healthyText={healthyText}
        labelMap={sensorUpgradeLabelMap}
        problemText={problemText}
        styleMap={sensorUpgradeStyleMap}
    />
);

export default SensorUpgrade;
