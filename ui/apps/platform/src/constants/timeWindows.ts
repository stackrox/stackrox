export const timeWindows = [
    'Past hour',
    'Past 8 hours',
    'Past day',
    'Past week',
    'Past month',
    'All time',
] as const;

export type TimeWindow = (typeof timeWindows)[number];

export const snoozeDurations = {
    DAY: '1 Day',
    WEEK: '1 Week',
    MONTH: '1 Month',
    UNSET: 'Indefinite',
} as const;

export const durations = {
    HOUR: '3600s',
    DAY: '86400s', // 24 * 3600 seconds
    WEEK: '604800s', // 7 * 24 * 3600 seconds
    MONTH: '2592000s', // 30 * 24 * 3600 seconds
    UNSET: '0',
} as const;
