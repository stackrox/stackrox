import type { ReactElement } from 'react';

import type { ClusterHealthStatus } from 'types/cluster.proto';

import { healthStatusLabels } from '../cluster.constants';
import { healthStatusStyles } from '../cluster.helpers';
import HealthStatus from './HealthStatus';

type ClusterStatusProps = {
    healthStatus?: ClusterHealthStatus;
};

function ClusterStatus({ healthStatus }: ClusterStatusProps): ReactElement {
    const { overallHealthStatus = 'UNAVAILABLE' } = healthStatus ?? {};

    const { Icon, fgColor } = healthStatusStyles[overallHealthStatus];
    const icon = <Icon />;

    return (
        <HealthStatus icon={icon} iconColor={fgColor}>
            {healthStatusLabels[overallHealthStatus]}
        </HealthStatus>
    );
}

export default ClusterStatus;
