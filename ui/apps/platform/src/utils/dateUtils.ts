import Raven from 'raven-js';
import { distanceInWordsStrict } from 'date-fns';

export type DateLike = string | number | Date;

const defaultDateFormatOptions = {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
} as const;

const defaultTimeFormatOptions = {
    hour: 'numeric',
    minute: 'numeric',
    second: 'numeric',
} as const;

const defaultDateTimeFormatOptions = {
    ...defaultDateFormatOptions,
    ...defaultTimeFormatOptions,
    timeZoneName: 'short',
} as const;

function convertDateLikeToDate(dateLike: DateLike): Date {
    const dateOrTimestamp = typeof dateLike === 'string' ? Date.parse(dateLike) : dateLike;
    return new Date(dateOrTimestamp);
}

function formatLocalizedDateTime(
    dateLike: DateLike,
    locales: Intl.LocalesArgument = undefined,
    dateTimeFormatOptions: Intl.DateTimeFormatOptions = {}
): string {
    const date = convertDateLikeToDate(dateLike);
    if (date.toString() === 'Invalid Date') {
        Raven.captureException(new Error(`Invalid date: ${String(dateLike)}`));
        return String(dateLike);
    }
    return new Intl.DateTimeFormat(locales, {
        ...dateTimeFormatOptions,
    }).format(date);
}

/**
 * Formats a date in ISO 8601 standard
 * @param dateLike - A timestamp, formatted date string, or Date object
 * @returns An ISO 8601 string representation of the date
 */
export function displayDateTimeAsISO8601(dateLike: DateLike) {
    const date = convertDateLikeToDate(dateLike);
    if (date.toString() === 'Invalid Date') {
        Raven.captureException(new Error(`Invalid date: ${String(dateLike)}`));
        return String(dateLike);
    }
    return date.toISOString();
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
    locales: Intl.LocalesArgument = undefined,
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
    locales: Intl.LocalesArgument = undefined,
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
    locales: Intl.LocalesArgument = undefined,
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
    locales: Intl.LocalesArgument = undefined,
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
