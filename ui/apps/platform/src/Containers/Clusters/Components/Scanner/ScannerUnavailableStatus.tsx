import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import HealthStatus from '../HealthStatus';

type ScannerUnavailableStatusProps = {
    isList?: boolean;
    icon: ReactElement;
    fgColor: string;
    healthStatusElement: ReactElement;
    healthLabelElement: ReactElement;
};

const ScannerUnavailableStatus = ({
    isList = false,
    icon,
    fgColor,
    healthStatusElement,
    healthLabelElement,
}: ScannerUnavailableStatusProps) => {
    const reasonUnavailable = (
        <div data-testid="scannerInfoComplete">
            <strong>Upgrade Sensor</strong> to get Scanner health information
        </div>
    );

    return isList ? (
        <Tooltip content={reasonUnavailable}>
            <div className="inline">{healthStatusElement}</div>
        </Tooltip>
    ) : (
        <HealthStatus icon={icon} iconColor={fgColor}>
            <div>
                {healthLabelElement}
                {reasonUnavailable}
            </div>
        </HealthStatus>
    );
};

export default ScannerUnavailableStatus;
