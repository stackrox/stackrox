import { getQueryString, startingTimeRegExp } from './diagnosticBundleUtils';

describe('Diagnostic Bundle dialog box', () => {
    describe('query string', () => {
        it('should not have params for initial defaults', () => {
            const expected = '';
            const received = getQueryString({
                selectedClusterNames: [],
                isStartingTimeValid: true,
                startingTimeObject: null,
            });

            expect(received).toBe(expected);
        });

        it('should have param for one selected cluster', () => {
            const expected = '?cluster=abbot';
            const received = getQueryString({
                selectedClusterNames: ['abbot'],
                isStartingTimeValid: true,
                startingTimeObject: null,
            });

            expect(received).toBe(expected);
        });

        // qs encodeValuesOnly option replaces colon with %3A in starting time.

        it('should have a param for valid starting time', () => {
            const expected = '?since=2020-10-20T20%3A21%3A00.000Z';
            const received = getQueryString({
                selectedClusterNames: [],
                isStartingTimeValid: true,
                startingTimeObject: new Date('2020-10-20T20:21Z'), // seconds are optional
            });

            expect(received).toBe(expected);
        });

        it('should have params for one selected cluster and valid starting time', () => {
            const expected = '?cluster=costello&since=2020-10-20T20%3A21%3A22.000Z';
            const received = getQueryString({
                selectedClusterNames: ['costello'],
                isStartingTimeValid: true,
                startingTimeObject: new Date('2020-10-20T20:21:22Z'), // thousandths are optional
            });

            expect(received).toBe(expected);
        });

        it('should have params for two selected clusters and valid starting time', () => {
            const expected = '?cluster=costello&cluster=abbot&since=2020-10-20T20%3A21%3A22.345Z';
            const received = getQueryString({
                selectedClusterNames: ['costello', 'abbot'],
                isStartingTimeValid: true,
                startingTimeObject: new Date('2020-10-20T20:21:22.345Z'),
            });

            expect(received).toBe(expected);
        });
    });

    describe('starting time format', () => {
        it('should not match empty string', () => {
            const startingTimeText = ''; // represents default starting time

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match default stringification', () => {
            const startingTimeText = 'Tue Oct 20 2020 17:22:00 GMT-0400';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match application stringification', () => {
            const startingTimeText = '10/20/2020 17:22:00';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match numeric value as string', () => {
            const startingTimeText = '1603228920000';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match incomplete yyyy-mm-dd even with time zone', () => {
            const startingTimeText = '2020-10-20Z';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match yyyy-mm-ddThh:mm without time zone', () => {
            const startingTimeText = '2020-10-20T21:22';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should not match yyyy-mm-ddThh:mm with a different time zone than UTC', () => {
            const startingTimeText = '2020-10-20T21:22-04';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(false);
        });

        it('should match yyyy-mm-ddThh:mmZ without seconds', () => {
            const startingTimeText = '2020-10-20T21:22Z';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(true);
        });

        it('should match yyyy-mm-ddThh:mm:ssZ without thousandths', () => {
            const startingTimeText = '2020-10-20T21:22:23Z';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(true);
        });

        it('should match yyyy-mm-ddThh:mm:ss.tttZ', () => {
            const startingTimeText = '2020-10-20T21:22:23.456Z';

            expect(startingTimeRegExp.test(startingTimeText)).toBe(true);
        });
    });
});
