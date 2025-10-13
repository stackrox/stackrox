import React from 'react';
import type { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';
import pluralize from 'pluralize';

import { getTimeHoursMinutes } from 'utils/dateUtils';

type UpdatedTimeOrUpdateButtonProps = {
    countAvailable: number;
    isAvailableEqualToPerPage: boolean;
    isDisabled: boolean;
    lastUpdatedTime: string; // ISO 8601
    updateEvents: () => void;
};

const UpdatedTimeOrUpdateButton = ({
    countAvailable,
    isAvailableEqualToPerPage,
    isDisabled,
    lastUpdatedTime,
    updateEvents,
}: UpdatedTimeOrUpdateButtonProps): ReactElement => {
    return countAvailable === 0 ? (
        <em className="pf-v5-u-font-size-sm pf-v5-u-text-nowrap">{`Last updated at ${getTimeHoursMinutes(
            lastUpdatedTime
        )}`}</em>
    ) : (
        <Button isDisabled={isDisabled} size="sm" onClick={updateEvents} variant="secondary">
            {`${countAvailable}${isAvailableEqualToPerPage ? '+' : ''} ${pluralize(
                'event',
                countAvailable
            )} available`}
        </Button>
    );
};

export default UpdatedTimeOrUpdateButton;
