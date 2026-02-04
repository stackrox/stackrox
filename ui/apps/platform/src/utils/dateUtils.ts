import Raven from 'raven-js';
import { distanceInWordsStrict } from 'date-fns';

import type { Schedule, ScheduleBase } from 'types/schedule.proto';

const userLanguages: readonly string[] | undefined = globalThis.navigator?.languages;

export type DateLike = string | number | Date;

const defaultDateFormatOptions: Readonly<Intl.DateTimeFormatOptions> = {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
};

const defaultTimeFormatOptions: Readonly<Intl.DateTimeFormatOptions> = {
    hour: 'numeric',
    minute: 'numeric',
    second: 'numeric',
};

const defaultDateTimeFormatOptions: Readonly<Intl.DateTimeFormatOptions> = {
    ...defaultDateFormatOptions,
    ...defaultTimeFormatOptions,
    timeZoneName: 'short',
};

function convertDateLikeToDate(dateLike: DateLike): Date {
    const dateOrTimestamp = typeof dateLike === 'string' ? Date.parse(dateLike) : dateLike;
    return new Date(dateOrTimestamp);
}

function formatLocalizedDateTime(
    dateLike: DateLike,
    locales: Intl.LocalesArgument = userLanguages,
    dateTimeFormatOptions: Intl.DateTimeFormatOptions = {}
): string {
    try {
        const preferredLocale: string | Intl.Locale | undefined = Array.isArray(locales)
            ? locales[0]
            : locales;
        const date = convertDateLikeToDate(dateLike);
        return new Intl.DateTimeFormat(preferredLocale, {
            ...dateTimeFormatOptions,
        }).format(date);
    } catch (e: unknown) {
        Raven.captureException(e);
        return String(dateLike);
    }
}

/**
 * Returns a human readable label for a recurring schedule.
 * @param schedule - A `Schedule` object describing the interval and time
 * @returns A formatted string such as "Daily at 05:00 UTC" or "Every Mon and Wed at 13:30 UTC"
 */
export function formatRecurringSchedule(schedule: Schedule) {
    const formatDays = (days: string[]): string => {
        if (days.length === 1) {
            return days[0];
        }
        if (days.length === 2) {
            return days.join(' and ');
        }
        return `${days.slice(0, -1).join(', ')}, and ${days[days.length - 1]}`;
    };

    const timeString = `${getHourMinuteStringFromScheduleBase(schedule)} UTC`;

    switch (schedule.intervalType) {
        case 'DAILY':
            return `Daily at ${timeString}`;
        case 'WEEKLY': {
            const daysOfWeek = schedule.daysOfWeek.days.map((day) => daysOfWeekAbbreviated[day]);
            return `Every ${formatDays(daysOfWeek)} at ${timeString}`;
        }
        case 'MONTHLY': {
            const formattedDaysOfMonth = schedule.daysOfMonth.days.map(getDayOfMonthWithOrdinal);
            return `Monthly on the ${formatDays(formattedDaysOfMonth)} at ${timeString}`;
        }
        default:
            return 'Invalid Schedule';
    }
}

function padStart2(timeElement: number) {
    return timeElement.toString().padStart(2, '0');
}

export function getHourMinuteStringFromScheduleBase({ hour, minute }: ScheduleBase) {
    // Return 24-hour hh:mm string for hour and minute.
    return [padStart2(hour), padStart2(minute)].join(':');
}

/**
 * Formats a date in ISO 8601 standard
 * @param dateLike - A timestamp, formatted date string, or Date object
 * @returns An ISO 8601 string representation of the date
 */
export function displayDateTimeAsISO8601(dateLike: DateLike) {
    try {
        const date = convertDateLikeToDate(dateLike);
        return date.toISOString();
    } catch (e: unknown) {
        Raven.captureException(e);
        return String(dateLike);
    }
}

/**
 * Returns a formatted date and time, defaulted to the current locale's format
 * @param dateLike - A timestamp, formatted date string, or Date object
 * @param locales - THe locale or locales to use for formatting. `undefined` uses the browsers current locale
 * @param dateTimeFormatOptionOverrides - Format override options for the returned datetime string
 * @returns {string} - returns a formatted string for the date time
 */
export function getDateTime(
    dateLike: DateLike,
    locales: Intl.LocalesArgument = userLanguages,
    dateTimeFormatOptionOverrides: Intl.DateTimeFormatOptions = {}
) {
    return formatLocalizedDateTime(dateLike, locales, {
        ...defaultDateTimeFormatOptions,
        ...dateTimeFormatOptionOverrides,
    });
}

/**
 * Returns a formatted date
 * @param timestamp - The timestamp for the date
 * @param locales - THe locale or locales to use for formatting. `undefined` uses the browsers current locale
 * @param dateTimeFormatOptionOverrides - Format override options for the returned datetime string
 * @returns returns a formatted string for the date
 */
export function getDate(
    dateLike: DateLike,
    locales: Intl.LocalesArgument = userLanguages,
    dateTimeFormatOptionOverrides: Intl.DateTimeFormatOptions = {}
): string {
    return formatLocalizedDateTime(dateLike, locales, {
        ...defaultDateFormatOptions,
        ...dateTimeFormatOptionOverrides,
    });
}

/**
 * Returns a formatted time
 * @param timestamp - The timestamp for the date
 * @param locales - THe locale or locales to use for formatting. `undefined` uses the browsers current locale
 * @param dateTimeFormatOptionOverrides - Format override options for the returned datetime string
 * @returns - returns a formatted string for the time
 */
export function getTime(
    dateLike: DateLike,
    locales: Intl.LocalesArgument = userLanguages,
    dateTimeFormatOptionOverrides: Intl.DateTimeFormatOptions = {}
): string {
    return formatLocalizedDateTime(dateLike, locales, {
        ...defaultTimeFormatOptions,
        ...dateTimeFormatOptionOverrides,
    });
}

/**
 * Returns a formatted time with hours and minutes but without seconds.
 * @param timestamp - The timestamp for the date
 * @param locales - THe locale or locales to use for formatting. `undefined` uses the browsers current locale
 * @param dateTimeFormatOptionOverrides - Format override options for the returned datetime string
 * @returns - returns a formatted string for the time
 */
export function getTimeHoursMinutes(
    timestamp: DateLike,
    locales: Intl.LocalesArgument = userLanguages,
    dateTimeFormatOptionOverrides: Intl.DateTimeFormatOptions = {}
): string {
    return getTime(timestamp, locales, {
        hour: 'numeric',
        second: undefined,
        ...dateTimeFormatOptionOverrides,
    });
}

export function addBrandedTimestampToString(str: string) {
    return `StackRox:${str}-${getDate(new Date())}`;
}

const daysOfWeek = [
    'Sunday',
    'Monday',
    'Tuesday',
    'Wednesday',
    'Thursday',
    'Friday',
    'Saturday',
] as const;

const daysOfWeekAbbreviated = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const;

/**
 * Given an ISO 8601 string, return the day of the week.
 *
 * date-fns@2: replace new Date(timestamp).getDay() with getDay(parseISO(timestamp))
 */
export const getDayOfWeek = (timestamp: DateLike) => daysOfWeek[new Date(timestamp).getDay()];

/*
 * Given an ISO 8601 string and Date instance, return the time difference.
 *
 * Specify rounding method explicitly because default changes to 'round' in date-fns@2.
 * formatDistanceStrict(currentDatetime, parseISO(dataDatetime), { roundingMethod: 'floor' });
 */
export const getDistanceStrict: typeof distanceInWordsStrict = (
    dataDatetime,
    currentDatetime,
    options
) => distanceInWordsStrict(dataDatetime, currentDatetime, options);
/*
 * Given an ISO 8601 string and Date instance, return the time difference:
 * if currentDatetime is in X units
 * if currentDatetime is X units ago
 *
 * Specify rounding method explicitly because default changes to 'round' in date-fns@2.
 * Also the order of the arguments is reversed in date-fns@2
 * formatDistanceStrict(parseISO(dataDatetime), currentDatetime, { roundingMethod: 'floor', addSuffix: true });
 */
export const getDistanceStrictAsPhrase = (
    dataDatetime: DateLike,
    currentDatetime: DateLike,
    unit?: 's' | 'm' | 'h' | 'd' | 'M' | 'Y'
) =>
    distanceInWordsStrict(currentDatetime, dataDatetime, {
        addSuffix: true,
        partialMethod: 'floor',
        unit,
    });

/**
 * Returns day of month with its ordinal suffix.
 * @param {number} num - The number to get the ordinal suffix for. Confirmed to work with 1 through 31.
 * @returns {string} - The number with its ordinal suffix (e.g., "1st", "2nd", "3rd", "4th", etc.)
 */
export function getDayOfMonthWithOrdinal(num: number): string {
    if (num === 11 || num === 12 || num === 13) {
        return `${num}th`;
    }

    switch (num % 10) {
        case 1:
            return `${num}st`;
        case 2:
            return `${num}nd`;
        case 3:
            return `${num}rd`;
        default:
            return `${num}th`;
    }
}
