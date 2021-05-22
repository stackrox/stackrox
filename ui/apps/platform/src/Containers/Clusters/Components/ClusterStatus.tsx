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
    const icon = <Icon className={`${isList ? 'inline' : ''} h-4 w-4`} />;
    return (
        <div>
            <div className={`${isList ? 'mb-1' : ''}`}>
                <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
                    <div data-testid="clusterStatus" className={`${isList ? 'inline' : ''}`}>
                        <span className={`${bgColor} ${fgColor}`}>
                            {healthStatusLabels[overallHealthStatus]}
                        </span>
                    </div>
                </HealthStatus>
            </div>
            {isList && <ClusterStatusPill healthStatus={healthStatus} />}
        </div>
    );
}

export default ClusterStatus;
