import type { ReactElement } from 'react';
import { Flex, FlexItem, Icon } from '@patternfly/react-core';
import {
    BanIcon,
    CheckIcon,
    ExclamationTriangleIcon,
    InfoCircleIcon,
} from '@patternfly/react-icons';
import { distanceInWordsStrict } from 'date-fns';

const iconDefault = (
    <Icon>
        <InfoCircleIcon color="var(--pf-v5-global--info-color--100)" />
    </Icon>
);

const iconValid = (
    <Icon>
        <CheckIcon color="var(--pf-v5-global--success-color--100)" />
    </Icon>
);

const iconFuture = (
    <Icon>
        <BanIcon color="var(--pf-v5-global--danger-color--100)" />
    </Icon>
); // danger because it time is complete and incorrect

const iconInvalid = (
    <Icon>
        <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />
    </Icon>
); // warning because time might be incomplete

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
    let icon: ReactElement = iconDefault;
    let message = 'default time: 20 minutes ago';

    if (currentTimeObject && startingTimeObject) {
        const timeDifference = distanceInWordsStrict(currentTimeObject, startingTimeObject, {
            partialMethod: 'round',
        });

        if (isStartingTimeValid) {
            icon = iconValid;
            message = `about ${timeDifference} ago`;
        } else {
            icon = iconFuture;
            message = `future time: in about ${timeDifference}`;
        }
    } else if (!isStartingTimeValid) {
        icon = iconInvalid;
        message = `expected format: ${startingTimeFormat}`;
    }

    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            {icon}
            <FlexItem data-testid="starting-time-message">{message}</FlexItem>
        </Flex>
    );
};

export default FilterByStartingTimeValidationMessage;
