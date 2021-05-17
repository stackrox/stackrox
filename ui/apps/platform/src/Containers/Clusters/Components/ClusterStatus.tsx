import React, { ReactElement } from 'react';

import { healthStatusLabels } from 'messages/common';
import HealthStatus from './HealthStatus';
import ClusterStatusPill from './ClusterStatusPill';
import { healthStatusStyles } from '../cluster.helpers';
import { ClusterHealthStatus } from '../clusterTypes';

/*
 * Cluster Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */

type ClusterStatusProps = {
    healthStatus: ClusterHealthStatus;
    isList?: boolean;
};

function ClusterStatus({ healthStatus, isList = false }: ClusterStatusProps): ReactElement {
    const { overallHealthStatus } = healthStatus;
    const { Icon, bgColor, fgColor } = healthStatusStyles[overallHealthStatus];
    const icon = <Icon className="h-4 w-4" />;
    return (
        <div>
            <HealthStatus icon={icon} iconColor={fgColor}>
                <div data-testid="clusterStatus">
                    <span className={`${bgColor} ${fgColor}`}>
                        {healthStatusLabels[overallHealthStatus]}
                    </span>
                </div>
            </HealthStatus>
            {isList && <ClusterStatusPill healthStatus={healthStatus} />}
        </div>
    );
}

export default ClusterStatus;
