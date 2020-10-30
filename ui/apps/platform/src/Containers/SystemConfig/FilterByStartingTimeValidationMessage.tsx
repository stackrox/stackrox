import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';
import { AlertTriangle, Check, Info, X } from 'react-feather';
import { distanceInWordsStrict } from 'date-fns';

import HealthStatus from 'Containers/Clusters/Components/HealthStatus';

// Dislay validation and information in similar format to cluster health.
const FilterByStartingTimeValidationMessage = ({
    currentTimeObject,
    isStartingTimeValid,
    startingTimeFormat,
    startingTimeObject,
}): ReactElement => {
    let classNameColor = 'text-primary-700';
    let Icon = Info;
    let message = 'default time: 20 minutes ago';

    if (startingTimeObject) {
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

FilterByStartingTimeValidationMessage.propTypes = {
    currentTimeObject: PropTypes.instanceOf(Date),
    isStartingTimeValid: PropTypes.bool.isRequired,
    startingTimeFormat: PropTypes.string.isRequired,
    startingTimeObject: PropTypes.instanceOf(Date),
};

FilterByStartingTimeValidationMessage.defaultProps = {
    currentTimeObject: null,
    startingTimeObject: null,
} as FilterByStartingTimeValidationMessageProps;

export type FilterByStartingTimeValidationMessageProps = InferProps<
    typeof FilterByStartingTimeValidationMessage.propTypes
>;
export default FilterByStartingTimeValidationMessage;
