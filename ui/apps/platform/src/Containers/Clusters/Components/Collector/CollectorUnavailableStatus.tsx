import React, { ReactElement } from 'react';

import { Tooltip, TooltipOverlay } from '@stackrox/ui-components';
import HealthStatus from '../HealthStatus';

type CollectorUnavailableStatusProps = {
    isList?: boolean;
    icon: ReactElement;
    fgColor: string;
    statusElement: ReactElement;
};

function CollectorUnavailableStatus({
    isList = false,
    icon,
    fgColor,
    statusElement,
}: CollectorUnavailableStatusProps): ReactElement {
    const reasonUnavailable = (
        <div data-testid="collectorInfoComplete">
            <strong>Upgrade Sensor</strong> to get Collector health information
        </div>
    );

    return isList ? (
        <Tooltip content={<TooltipOverlay>{reasonUnavailable}</TooltipOverlay>}>
            <div>
                <HealthStatus icon={icon} iconColor={fgColor}>
                    {statusElement}
                </HealthStatus>
            </div>
        </Tooltip>
    ) : (
        <HealthStatus icon={icon} iconColor={fgColor}>
            <div>
                {statusElement}
                {reasonUnavailable}
            </div>
        </HealthStatus>
    );
}

export default CollectorUnavailableStatus;
