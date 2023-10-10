import { distanceInWordsStrict, format, addDays } from 'date-fns';

import dateTimeFormat, { dateFormat, timeFormat } from 'constants/dateTimeFormat';
import { IntervalType } from 'types/report.proto';

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
    return format(timestamp, 'h:mmA');
}

export function getLatestDatedItemByKey<T>(key: string | null, list: T[] = []): T | null {
    if (!key || !list.length || !list[0][key]) {
        return null;
    }

    return list.reduce((acc: T | null, item) => {
        const nextDate = item[key] && Date.parse(item[key]);

        if (!acc || nextDate > Date.parse(acc[key])) {
            return item;
        }

        return acc;
    }, null);
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
export const getDistanceStrictAsPhrase = (dataDatetime: DateLike, currentDatetime: DateLike) =>
    distanceInWordsStrict(currentDatetime, dataDatetime, {
        addSuffix: true,
        partialMethod: 'floor',
    });

export const addDaysToDate = (date: DateLike, amount: number) => {
    return format(addDays(date, amount + 1), 'YYYY-MM-DD[T]HH:mm:ss.SSSSSSSSS[Z]');
};

const weekDays = [
    { key: 1, dayName: 'Monday' },
    { key: 2, dayName: 'Tuesday' },
    { key: 3, dayName: 'Wednesday' },
    { key: 4, dayName: 'Thursday' },
    { key: 5, dayName: 'Friday' },
    { key: 6, dayName: 'Saturday' },
    { key: 0, dayName: 'Sunday' },
];

const monthDays = [
    { key: 1, dayName: 'the first of the month' },
    { key: 15, dayName: 'the middle of the month' },
];

// TODO The type of `days` is always `number[]` but it can't be annotated here due
// to some type mismatches elsewhere.
export function getDayList(dayListType: IntervalType, days) {
    const dayNameConstants = dayListType === 'WEEKLY' ? weekDays : monthDays;

    const dayNameArray = dayNameConstants.reduce((acc: string[], constant) => {
        const newItem = days.find((day) => day === constant.key);

        return typeof newItem !== 'undefined' ? [...acc, constant.dayName] : [...acc];
    }, []);

    return dayNameArray;
}

export default {
    getLatestDatedItemByKey,
    addBrandedTimestampToString,
};
