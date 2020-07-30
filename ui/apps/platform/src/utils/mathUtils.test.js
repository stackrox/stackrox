import { getPercentage } from './mathUtils';

describe('dateUtils', () => {
    describe('getPercentage', () => {
        it('should return 0 when total not present', () => {
            const number = 42;

            const percentage = getPercentage(number);

            expect(percentage).toEqual(0);
        });

        it('should return percentage one number is of second argument', () => {
            const number = 33;
            const total = 165;

            const percentage = getPercentage(number, total);

            expect(percentage).toEqual(20);
        });
    });
});
