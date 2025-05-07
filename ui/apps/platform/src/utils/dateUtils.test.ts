import {
    addBrandedTimestampToString,
    displayDateTimeAsISO8601,
    getDateTime,
    getDate,
    getTime,
    getTimeHoursMinutes,
    getDayOfMonthWithOrdinal,
    getDayOfWeek,
    getDistanceStrictAsPhrase,
} from './dateUtils';

describe('dateUtils', () => {
    describe('addBrandedTimestampToString', () => {
        it('should return string with branding prepended, and current data appended', () => {
            const currentDate = new Date();
            const month = `0${currentDate.getMonth() + 1}`.slice(-2);
            const dayOfMonth = `0${currentDate.getDate()}`.slice(-2);
            const year = currentDate.getFullYear();

            const baseName = `Vulnerability Management CVES Report`;

            const fileName = addBrandedTimestampToString(baseName);

            expect(fileName).toEqual(`StackRox:${baseName}-${month}/${dayOfMonth}/${year}`);
        });
    });
});

describe('displayDateTimeAsISO8601', () => {
    it('should return compliant ISO string with Z', () => {
        const date = new Date('2024-01-04T12:34:56Z');
        const result = displayDateTimeAsISO8601(date);
        expect(result).toBe('2024-01-04T12:34:56.000Z');
    });
});

describe('getDateTime', () => {
    it('should format a datetime string with timezone', () => {
        const result = getDateTime(new Date('2024-01-04T12:34:56Z'), 'en-US');
        expect(result).toBe('01/04/2024, 12:34:56 PM UTC');
    });
});

describe('getDate', () => {
    it('should format only the date portion', () => {
        const result = getDate(new Date('2024-01-04T00:00:00Z'), 'en-US');
        expect(result).toBe('01/04/2024');
    });
});

describe('getTime', () => {
    it('should format only the time portion', () => {
        const result = getTime(new Date('2024-01-04T13:45:30Z'), 'en-US');
        expect(result).toBe('1:45:30 PM');
    });
});

describe('getTimeHoursMinutes', () => {
    it('should return only hours and minutes', () => {
        const result = getTimeHoursMinutes(new Date('2024-01-04T08:22:59Z'), 'en-US');
        expect(result).toBe('8:22 AM');
    });
});

describe('getDayOfMonthWithOrdinal', () => {
    it('should return ordinal day of month', () => {
        expect(getDayOfMonthWithOrdinal(1)).toBe('1st');
        expect(getDayOfMonthWithOrdinal(2)).toBe('2nd');
        expect(getDayOfMonthWithOrdinal(3)).toBe('3rd');
        expect(getDayOfMonthWithOrdinal(4)).toBe('4th');
        expect(getDayOfMonthWithOrdinal(10)).toBe('10th');
        expect(getDayOfMonthWithOrdinal(11)).toBe('11th');
        expect(getDayOfMonthWithOrdinal(12)).toBe('12th');
        expect(getDayOfMonthWithOrdinal(13)).toBe('13th');
        expect(getDayOfMonthWithOrdinal(14)).toBe('14th');
        expect(getDayOfMonthWithOrdinal(20)).toBe('20th');
        expect(getDayOfMonthWithOrdinal(21)).toBe('21st');
        expect(getDayOfMonthWithOrdinal(22)).toBe('22nd');
        expect(getDayOfMonthWithOrdinal(23)).toBe('23rd');
        expect(getDayOfMonthWithOrdinal(24)).toBe('24th');
    });
});

describe('getDayOfWeek', () => {
    it('should return localized weekday name', () => {
        expect(getDayOfWeek(new Date('2025-05-01T00:01:00Z'))).toBe('Thursday');
        expect(getDayOfWeek(new Date('2025-05-02T00:01:00Z'))).toBe('Friday');
        expect(getDayOfWeek(new Date('2025-05-03T00:01:00Z'))).toBe('Saturday');
        expect(getDayOfWeek(new Date('2025-05-04T00:01:00Z'))).toBe('Sunday');
        expect(getDayOfWeek(new Date('2025-05-05T00:01:00Z'))).toBe('Monday');
        expect(getDayOfWeek(new Date('2025-05-06T00:01:00Z'))).toBe('Tuesday');
        expect(getDayOfWeek(new Date('2025-05-07T00:01:00Z'))).toBe('Wednesday');
    });
});

describe('getDistanceStrictAsPhrase', () => {
    it('should return a human-readable phrase for time distance', () => {
        expect(getDistanceStrictAsPhrase(new Date().getTime() - 3600 * 1000, new Date())).toBe(
            '1 hour ago'
        );
        expect(
            getDistanceStrictAsPhrase(new Date().getTime() - 3600 * 1000 * 24 * 30, new Date())
        ).toBe('1 month ago');
        expect(
            getDistanceStrictAsPhrase(new Date().getTime() - 3600 * 1000 * 24 * 365, new Date())
        ).toBe('1 year ago');
        expect(
            getDistanceStrictAsPhrase(new Date().getTime() - 3600 * 1000 * 24 * 365 * 2, new Date())
        ).toBe('2 years ago');
    });
});
