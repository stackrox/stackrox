export const timeWindows = [
    'Past hour',
    'Past 8 hours',
    'Past day',
    'Past week',
    'Past month',
    'All time',
] as const;

export const snoozeDurations = {
    DAY: '1 Day',
    WEEK: '1 Week',
    MONTH: '1 Month',
    UNSET: 'Indefinite',
} as const;

export const durations = {
    HOUR: '1h',
    DAY: '24h',
    WEEK: '168h',
    MONTH: '720h',
    UNSET: '0',
} as const;
