import { distanceInWordsStrict, format } from 'date-fns';

const timeFormat = 'h:mm:ssA';
const dateFormat = 'MM/DD/YYYY';
const dateTimeFormat = `${dateFormat} | ${timeFormat}`;

type DateLike = string | number | Date;

/**
 * Returns a formatted date and time
 * @param {DateLike} timestamp - The timestamp for the date and time
 * @returns {string} - returns a formatted string for the date time
 */
export function getDateTime(timestamp: DateLike) {
    return format(timestamp, dateTimeFormat);
}

/**
 * Returns a formatted date
 * @param {DateLike} timestamp - The timestamp for the date
 * @returns {string} - returns a formatted string for the date
 */
export function getDate(timestamp: DateLike) {
    return format(timestamp, dateFormat);
}

/**
 * Returns a formatted time
 * @param {DateLike} timestamp - The timestamp for the date
 * @returns {string} - returns a formatted string for the time
 */
export function getTime(timestamp: DateLike) {
    return format(timestamp, timeFormat);
}

/**
 * Returns a formatted time with hours and minutes but without seconds.
 * @param {DateLike} timestamp - The timestamp for the date
 * @returns {string} - returns a formatted string for the time
 */
export function getTimeHoursMinutes(timestamp: DateLike) {
    return format(timestamp, 'h:mm A');
}

export function addBrandedTimestampToString(str: string) {
    return `StackRox:${str}-${format(new Date(), dateFormat)}`;
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
//
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
