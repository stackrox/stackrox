import React, { ReactElement } from 'react';

import { Schedule } from 'types/report.proto';
import { getDayList } from 'utils/dateUtils';

const intervalNamesMap = {
    WEEKLY: 'weekly',
    MONTHLY: 'monthly',
};

type ScheduleTextProps = {
    schedule: Schedule;
};

function ScheduleText({ schedule }: ScheduleTextProps): ReactElement {
    const intervalText = intervalNamesMap[schedule?.intervalType] || '';

    const dayListType = schedule?.intervalType === 'WEEKLY' ? 'daysOfWeek' : 'daysOfMonth';

    const dayList = getDayList(schedule?.intervalType, schedule[dayListType]?.days ?? []);

    // Intl.ListFormat is in ES2021
    //   rather than install a 3rd-party polyfill, if it's not present we fall back to just a comma-separated list
    let dayListText = '';
    try {
        // Intl.ListFormat support is not in TypeScript defs as of December 2021
        //   to track progress, see https://github.com/microsoft/TypeScript/issues/46907
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        dayListText = new Intl.ListFormat('en-US', { style: 'long', type: 'conjunction' }).format(
            dayList
        );
    } catch {
        dayListText = dayList.join(', ');
    }

    return (
        <span>
            Repeat report {intervalText} on {dayListText}
        </span>
    );
}

export default ScheduleText;
