import React, { ElementType, ReactElement } from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';
import {
    BanIcon,
    CheckIcon,
    ExclamationTriangleIcon,
    InfoCircleIcon,
} from '@patternfly/react-icons';
import { distanceInWordsStrict } from 'date-fns';

type Style = {
    Icon: ElementType;
    fgColor: string;
};

const styleDefault: Style = {
    Icon: InfoCircleIcon,
    fgColor: 'pf-u-info-color-100',
};

const styleValid: Style = {
    Icon: CheckIcon,
    fgColor: 'pf-u-success-color-100',
};

const styleFuture: Style = {
    Icon: BanIcon,
    fgColor: 'pf-u-danger-color-100', // alert because it time is complete and incorrect
};

const styleInvalid: Style = {
    Icon: ExclamationTriangleIcon,
    fgColor: 'pf-u-warning-color-100', // warning because time might be incomplete
};

type FilterByStartingTimeValidationMessageProps = {
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
}: FilterByStartingTimeValidationMessageProps): ReactElement => {
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
        <Flex alignItems={{ default: 'alignItemsCenter' }} className={fgColor}>
            <Icon />
            <FlexItem data-testid="starting-time-message">{message}</FlexItem>
        </Flex>
    );
};

export default FilterByStartingTimeValidationMessage;
