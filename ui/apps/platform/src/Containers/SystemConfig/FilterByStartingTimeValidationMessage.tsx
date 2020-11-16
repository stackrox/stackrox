import React, { ReactElement } from 'react';
import { AlertTriangle, Check, Info, X } from 'react-feather';
import { distanceInWordsStrict } from 'date-fns';

import HealthStatus from 'Containers/Clusters/Components/HealthStatus';

type Props = {
    currentTimeObject: Date | null;
    isStartingTimeValid: boolean;
    startingTimeFormat: string;
    startingTimeObject: Date | null;
};

// Dislay validation and information in similar format to cluster health.
const FilterByStartingTimeValidationMessage = ({
    currentTimeObject,
    isStartingTimeValid,
    startingTimeFormat,
    startingTimeObject,
}: Props): ReactElement => {
    let classNameColor = 'text-primary-700';
    let Icon = Info;
    let message = 'default time: 20 minutes ago';

    if (currentTimeObject && startingTimeObject) {
        const timeDifference = distanceInWordsStrict(currentTimeObject, startingTimeObject, {
            partialMethod: 'round',
        });

        if (isStartingTimeValid) {
            classNameColor = 'text-success-700';
            Icon = Check;
            message = `about ${timeDifference} ago`;
        } else {
            classNameColor = 'text-alert-700';
            Icon = X;
            message = `future time: in about ${timeDifference}`;
        }
    } else if (!isStartingTimeValid) {
        classNameColor = 'text-warning-700';
        Icon = AlertTriangle;
        message = `expected format: ${startingTimeFormat}`;
    }

    return (
        <HealthStatus Icon={Icon} iconColor={classNameColor}>
            <span className={classNameColor} data-testid="starting-time-message">
                {message}
            </span>
        </HealthStatus>
    );
};

export default FilterByStartingTimeValidationMessage;
