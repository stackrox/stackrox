import React, { ElementType, ReactElement } from 'react';
import { AlertTriangle, Check, Info, X } from 'react-feather';
import { distanceInWordsStrict } from 'date-fns';

type Style = {
    Icon: ElementType;
    fgColor: string;
};

const styleDefault: Style = {
    Icon: Info,
    fgColor: 'text-primary-700',
};

const styleValid: Style = {
    Icon: Check,
    fgColor: 'text-success-700',
};

const styleFuture: Style = {
    Icon: X,
    fgColor: 'text-alert-700', // alert because it time is complete and incorrect
};

const styleInvalid: Style = {
    Icon: AlertTriangle,
    fgColor: 'text-warning-700', // warning because time might be incomplete
};

type Props = {
    currentTimeObject: Date | null;
    isStartingTimeValid: boolean;
    startingTimeFormat: string;
    startingTimeObject: Date | null;
};

// Dislay validation and information.
const FilterByStartingTimeValidationMessage = ({
    currentTimeObject,
    isStartingTimeValid,
    startingTimeFormat,
    startingTimeObject,
}: Props): ReactElement => {
    let style = styleDefault;
    let message = 'default time: 20 minutes ago';

    if (currentTimeObject && startingTimeObject) {
        const timeDifference = distanceInWordsStrict(currentTimeObject, startingTimeObject, {
            partialMethod: 'round',
        });

        if (isStartingTimeValid) {
            style = styleValid;
            message = `about ${timeDifference} ago`;
        } else {
            style = styleFuture;
            message = `future time: in about ${timeDifference}`;
        }
    } else if (!isStartingTimeValid) {
        style = styleInvalid;
        message = `expected format: ${startingTimeFormat}`;
    }

    const { Icon, fgColor } = style;

    return (
        <div className={`flex flex-row items-start leading-normal ${fgColor}`}>
            <Icon className="flex-shrink-0 h-4 w-4" />
            <span className="ml-2" data-testid="starting-time-message">
                {message}
            </span>
        </div>
    );
};

export default FilterByStartingTimeValidationMessage;
