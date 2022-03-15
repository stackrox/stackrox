import { distanceInWordsStrict, format, addDays } from 'date-fns';

import dateTimeFormat, { dateFormat } from 'constants/dateTimeFormat';

/**
 * Returns a formatted date and time
 * @param {string | Date} timestamp - The timestamp for the date and time
 * @returns {string} - returns a formatted string for the date time
 */
export function getDateTime(timestamp) {
    return format(timestamp, dateTimeFormat);
}

/**
 * Returns a formatted date
 * @param {string | Date} timestamp - The timestamp for the date
 * @returns {string} - returns a formatted string for the date
 */
export function getDate(timestamp) {
    return format(timestamp, dateFormat);
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

export function addBrandedTimestampToString(str) {
    return `StackRox:${str as string}-${format(new Date(), dateFormat)}`;
}

const daysOfWeek = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/**
 * Given an ISO 8601 string, return the day of the week.
 *
 * date-fns@2: replace new Date(timestamp).getDay() with getDay(parseISO(timestamp))
 */
export const getDayOfWeek = (timestamp) => daysOfWeek[new Date(timestamp).getDay()];

/*
 * Given an ISO 8601 string and Date instance, return the time difference.
 *
 * Specify rounding method explicitly because default changes to 'round' in date-fns@2.
 * formatDistanceStrict(currentDatetime, parseISO(dataDatetime), { roundingMethod: 'floor' });
 */
export const getDistanceStrict = (dataDatetime, currentDatetime, options) =>
    distanceInWordsStrict(dataDatetime, currentDatetime, options);
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
export const getDistanceStrictAsPhrase = (dataDatetime, currentDatetime) =>
    distanceInWordsStrict(currentDatetime, dataDatetime, {
        addSuffix: true,
        partialMethod: 'floor',
    });

export const addDaysToDate = (date, amount: number) => {
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

export function getDayList(dayListType, days) {
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
