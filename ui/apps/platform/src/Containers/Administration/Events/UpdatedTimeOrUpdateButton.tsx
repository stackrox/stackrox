import React, { ReactElement } from 'react';
import { Button } from '@patternfly/react-core';
import pluralize from 'pluralize';

import { getTimeHoursMinutes } from 'utils/dateUtils';

type UpdatedTimeOrUpdateButtonProps = {
    countAvailable: number;
    isAvailableEqualToPerPage: boolean;
    lastUpdatedTime: string; // ISO 8601
    updateEvents: () => void;
};

const UpdatedTimeOrUpdateButton = ({
    countAvailable,
    isAvailableEqualToPerPage,
    lastUpdatedTime,
    updateEvents,
}: UpdatedTimeOrUpdateButtonProps): ReactElement => {
    return countAvailable === 0 ? (
        <em className="pf-u-font-size-sm pf-u-text-nowrap">{`Last updated at ${getTimeHoursMinutes(
            lastUpdatedTime
        )}`}</em>
    ) : (
        <Button isSmall onClick={updateEvents} variant="secondary">
            {`${countAvailable}${isAvailableEqualToPerPage ? '+' : ''} ${pluralize(
                'event',
                countAvailable
            )} available`}
        </Button>
    );
};

export default UpdatedTimeOrUpdateButton;
