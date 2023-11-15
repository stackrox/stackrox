import React, { ReactElement } from 'react';

import { DecommissionedClusterRetentionInfo } from 'types/clusterService.proto';

import HealthStatusNotApplicable from './HealthStatusNotApplicable';

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
        // const healthStatus = getClusterDeletionStatus(daysUntilDeletion);
        // TODO IconText with something like SystemHealth/CardHeaderIcons? But what about Not applicable? MinusIcon?
        /* eslint-disable no-nested-ternary */
        const text =
            daysUntilDeletion < 1
                ? 'Imminent'
                : daysUntilDeletion === 1
                  ? 'in 1 day'
                  : `in ${daysUntilDeletion} days`;
        /* eslint-enable no-nested-ternary */

        return <span>{text}</span>;
    }

    // Cluster will not be deleted even if sensor status remains UNHEALTHY, because it has an ignore label.
    return <span>Excluded from deletion</span>;
}

export default ClusterDeletion;
