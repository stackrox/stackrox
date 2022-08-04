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
        // Cluster does not have sensor status UNHEALTHY or cluster deletion is turned off.
        return <HealthStatusNotApplicable testId={testId} />;
    }

    // Adapt health status categories to cluster deletion.

    if ('daysUntilDeletion' in clusterRetentionInfo) {
        // Cluster will be deleted if sensor status remains UNHEALTHY for the number of days.
        const { daysUntilDeletion } = clusterRetentionInfo;
        const healthStatus = getClusterDeletionStatus(daysUntilDeletion);
        const { bgColor, fgColor } = healthStatusStyles[healthStatus];
        /* eslint-disable no-nested-ternary */
        const text =
            daysUntilDeletion < 1
                ? 'Imminent'
                : daysUntilDeletion === 1
                ? 'in 1 day'
                : `in ${daysUntilDeletion} days`;
        /* eslint-enable no-nested-ternary */

        return <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>{text}</span>;
    }

    // Cluster will not be deleted even if sensor status remains UNHEALTHY, because it has an ignore label.
    const { bgColor, fgColor } = healthStatusStyles.HEALTHY;
    return (
        <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>Excluded from deletion</span>
    );
}

export default ClusterDeletion;
