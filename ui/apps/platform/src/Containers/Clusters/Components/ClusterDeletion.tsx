import React, { ReactElement } from 'react';

import { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { getClusterDeletionStatus, healthStatusStyles } from '../cluster.helpers';

const testId = 'clusterDeletion';

type ClusterDeletionProps = {
    clusterRetentionInfo: DecommissionedClusterRetentionInfo;
};

function ClusterDeletion({ clusterRetentionInfo }: ClusterDeletionProps): ReactElement {
    if (clusterRetentionInfo === null) {
        // Cluster does not have sensor status UNHEALTHY.
        return <HealthStatusNotApplicable testId={testId} />;
    }

    // Adapt health status categories to cluster deletion.

    if ('daysUntilDeletion' in clusterRetentionInfo) {
        // Cluster will be deleted if sensor status remains UNHEALTHY for the number of days.
        const { daysUntilDeletion } = clusterRetentionInfo;
        const healthStatus = getClusterDeletionStatus(daysUntilDeletion);
        const { bgColor, fgColor } = healthStatusStyles[healthStatus];
        const text = daysUntilDeletion === 1 ? 'in 1 day' : `in ${daysUntilDeletion} days`;

        return <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>{text}</span>;
    }

    // Cluster will not be deleted even if sensor status remains UNHEALTHY:
    // because it has an ignore label, if true
    // because system configuration is never delete, if false
    const { bgColor, fgColor } = healthStatusStyles.HEALTHY;
    const { isExcluded } = clusterRetentionInfo;
    const text = isExcluded ? 'Excluded from deletion' : 'Deletion is turned off';

    return <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>{text}</span>;
}

export default ClusterDeletion;
