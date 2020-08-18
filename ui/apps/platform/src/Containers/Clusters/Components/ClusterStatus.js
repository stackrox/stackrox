import PropTypes from 'prop-types';
import React from 'react';

import { healthStatusLabels } from 'messages/common';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { healthStatusStyles } from '../cluster.helpers';

/*
 * Cluster Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */
const ClusterStatus = ({ overallHealthStatus }) => {
    if (overallHealthStatus) {
        const { Icon, bgColor, fgColor } = healthStatusStyles[overallHealthStatus];
        return (
            <HealthStatus Icon={Icon} iconColor={fgColor}>
                <div>
                    <span className={`${bgColor} ${fgColor}`}>
                        {healthStatusLabels[overallHealthStatus]}
                    </span>
                </div>
            </HealthStatus>
        );
    }

    return <HealthStatusNotApplicable />;
};

ClusterStatus.propTypes = {
    overallHealthStatus: PropTypes.oneOf(['UNINITIALIZED', 'UNHEALTHY', 'DEGRADED', 'HEALTHY']),
};

ClusterStatus.defaultProps = {
    overallHealthStatus: null,
};

export default ClusterStatus;
