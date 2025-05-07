import { subHours, subWeeks, subMonths } from 'date-fns';

import { TimeWindow } from 'constants/timeWindows';

function timeWindowToDate(timeWindow: TimeWindow): Date {
    const now = new Date();

    switch (timeWindow) {
        case 'Past hour':
            return subHours(now, 1);

        case 'Past 8 hours':
            return subHours(now, 8);

        case 'Past day':
            return subHours(now, 24);

        case 'Past week':
            return subWeeks(now, 1);

        case 'Past month':
            return subMonths(now, 1);

        case 'All time':
            return new Date(0);

        default:
            throw new Error('Unexpected time window');
    }
}

export function timeWindowToISO(timeWindow: TimeWindow): string {
    return timeWindowToDate(timeWindow).toISOString();
}
