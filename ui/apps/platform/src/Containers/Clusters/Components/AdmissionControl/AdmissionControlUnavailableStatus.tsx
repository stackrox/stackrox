import React, { ReactElement } from 'react';
import { Tooltip } from '@patternfly/react-core';

import HealthStatus from '../HealthStatus';

type AdmissionControlUnavailableStatusProps = {
    isList?: boolean;
    icon: ReactElement;
    fgColor: string;
    healthStatusElement: ReactElement;
    healthLabelElement: ReactElement;
};

function AdmissionControlUnavailableStatus({
    isList = false,
    icon,
    fgColor,
    healthStatusElement,
    healthLabelElement,
}: AdmissionControlUnavailableStatusProps): ReactElement {
    const reasonUnavailable = (
        <div data-testid="admissionControlInfoComplete">
            <strong>Upgrade Sensor</strong> to get Admission Control health information
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
}

export default AdmissionControlUnavailableStatus;
