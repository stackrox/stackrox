import type { ReactElement } from 'react';
import {
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
} from '@patternfly/react-core';

import type { Schedule } from 'types/schedule.proto';
import { formatRecurringSchedule } from 'utils/dateUtils';

// Generic component for use cases:
// Parameters of compliance scan schedule configuration
// Delivery of vulnerability report configuration

export type ScheduleDescriptionListGroupProps = {
    schedule: Schedule | null;
};

function ScheduleDescriptionListGroup({
    schedule,
}: ScheduleDescriptionListGroupProps): ReactElement {
    return (
        <DescriptionListGroup>
            <DescriptionListTerm>Schedule</DescriptionListTerm>
            <DescriptionListDescription>
                {schedule ? formatRecurringSchedule(schedule) : '-'}
            </DescriptionListDescription>
        </DescriptionListGroup>
    );
}

export default ScheduleDescriptionListGroup;
