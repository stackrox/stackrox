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
        return <HealthStatusNotApplicable testId={testId} />;
    }

    // Adapt health status categories to cluster deletion.

    if ('daysUntilDeletion' in clusterRetentionInfo) {
        const { daysUntilDeletion } = clusterRetentionInfo;
        const healthStatus = getClusterDeletionStatus(daysUntilDeletion);
        const { bgColor, fgColor } = healthStatusStyles[healthStatus];
        const text = daysUntilDeletion === 1 ? 'in 1 day' : `in ${daysUntilDeletion} days`;

        return <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>{text}</span>;
    }

    const { bgColor, fgColor } = healthStatusStyles.HEALTHY;
    const { isExcluded } = clusterRetentionInfo;
    const text = isExcluded ? 'Excluded from deletion' : 'Never delete';

    return <span className={`${bgColor} ${fgColor} whitespace-nowrap`}>{text}</span>;
}

export default ClusterDeletion;
