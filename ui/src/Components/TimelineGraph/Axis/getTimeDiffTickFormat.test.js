import getTimeDiffTickFormat from './getTimeDiffTickFormat';

const values = [0, 1, 2];

describe('getTimeDiffTickFormat', () => {
    it('should return nothing for the first tick value', () => {
        expect(getTimeDiffTickFormat(3600000, 0, values)).toEqual(null);
    });

    it('should return nothing for the last tick value', () => {
        expect(getTimeDiffTickFormat(3600000, 2, values)).toEqual(null);
    });

    it('should return the correct tick formats', () => {
        expect(getTimeDiffTickFormat(3600000 * 24 * 7 * 4 * 12, 1, values)).toEqual('+336days'); // ~1 year
        expect(getTimeDiffTickFormat(3600000 * 24 * 7 * 4, 1, values)).toEqual('+28days'); // ~1 month
        expect(getTimeDiffTickFormat(3600000 * 24 * 7, 1, values)).toEqual('+7days'); // 1 week
        expect(getTimeDiffTickFormat(3600000 * 24, 1, values)).toEqual('+1day'); // 1 day
        expect(getTimeDiffTickFormat(3600000, 1, values)).toEqual('+1:00h'); // 1 hour
        expect(getTimeDiffTickFormat(3600000 * 2, 1, values)).toEqual('+2:00h');
        expect(getTimeDiffTickFormat(3600000 / 2, 1, values)).toEqual('+30:00m');
        expect(getTimeDiffTickFormat(123456, 1, values)).toEqual('+2:03m');
        expect(getTimeDiffTickFormat(1000, 1, values)).toEqual('+1s');
        expect(getTimeDiffTickFormat(100, 1, values)).toEqual('+100ms');
        expect(getTimeDiffTickFormat(500, 1, values)).toEqual('+500ms');
    });
});
