import type { ReactElement } from 'react';
import { Flex, FlexItem, Text, TextContent, TextVariants, Title } from '@patternfly/react-core';

import { daysOfMonthMap, daysOfWeekMap } from 'Components/PatternFly/DayPickerDropdown';
import type { ReportFormValues } from '../forms/useReportFormValues';
import { commaSeparateWithAnd } from '../utils';

export type ScheduleDetailsProps = {
    formValues: ReportFormValues;
};

function ScheduleDetails({ formValues }: ScheduleDetailsProps): ReactElement {
    let interval = '';
    let days = '';

    if (formValues.schedule.intervalType === 'WEEKLY') {
        interval = 'week';
        const daysArr = formValues.schedule.daysOfWeek?.map((day) => daysOfWeekMap[day]) ?? [];
        days = commaSeparateWithAnd(daysArr);
    } else if (formValues.schedule.intervalType === 'MONTHLY') {
        interval = 'month';
        const daysArr =
            formValues.schedule.daysOfMonth?.map((day) => daysOfMonthMap[day].toLowerCase()) ?? [];
        days = commaSeparateWithAnd(daysArr);
    }

    let scheduleDetailsText = <span>No schedule set</span>;
    if (interval !== '' && days !== '') {
        scheduleDetailsText = (
            <span>
                Report is scheduled to be sent on <strong>{days}</strong> every{' '}
                <strong>{interval}</strong>
            </span>
        );
    }

    return (
        <Flex direction={{ default: 'column' }}>
            <FlexItem>
                <Title headingLevel="h2">Schedule details</Title>
            </FlexItem>
            <FlexItem flex={{ default: 'flexNone' }}>
                <TextContent>
                    <Text component={TextVariants.p}>{scheduleDetailsText}</Text>
                </TextContent>
            </FlexItem>
        </Flex>
    );
}

export default ScheduleDetails;
