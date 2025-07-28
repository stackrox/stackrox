import React, { ReactElement } from 'react';

import HealthStatus from './HealthStatus';
import HealthStatusNotApplicable from './HealthStatusNotApplicable';
import { findUpgradeState, sensorUpgradeStyles } from '../cluster.helpers';
import { SensorUpgradeStatus } from '../clusterTypes';

const testId = 'sensorUpgrade';

type SensorUpgradeProps = {
    upgradeStatus?: SensorUpgradeStatus;
};

function SensorUpgrade({ upgradeStatus }: SensorUpgradeProps): ReactElement {
    if (upgradeStatus) {
        const upgradeStateObject = findUpgradeState(upgradeStatus);
        if (upgradeStateObject) {
            const { displayValue, type } = upgradeStateObject;

            const { Icon, fgColor } = sensorUpgradeStyles[type];
            const icon = <Icon className="h-4 w-4" />;

            return (
                <HealthStatus icon={icon} iconColor={fgColor}>
                    <div data-testid={testId}>{displayValue}</div>
                </HealthStatus>
            );
        }
    }

    return <HealthStatusNotApplicable testId={testId} />;
}

export default SensorUpgrade;
