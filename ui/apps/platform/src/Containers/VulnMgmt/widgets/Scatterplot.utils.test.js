import { getHighValue, getLowValue } from './Scatterplot.utils';

describe('visuals.helpers', () => {
    describe('getHighValue', () => {
        it('should find the max data value', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getHighValue(data, 'x');

            expect(highX).toEqual(89);
        });

        it('should find the different max data value for a different key', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getHighValue(data, 'y');

            expect(highX).toEqual(777);
        });

        it('should find the next multiple higher than the max data value, when multiple supplied', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];
            const multiple = 5;

            const highX = getHighValue(data, 'x', multiple);

            expect(highX).toEqual(90);
        });

        it('should find the next multiple higher than the max data value, when multiple supplied, plus padding if flag set', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];
            const multiple = 5;
            const shouldPad = true;

            const highX = getHighValue(data, 'x', multiple, shouldPad);

            expect(highX).toEqual(95);
        });

        it('should find the next multiple higher than the max data value, even for distant multiple', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];
            const multiple = 100;

            const highX = getHighValue(data, 'y', multiple);

            expect(highX).toEqual(800);
        });
    });

    describe('getLowValue', () => {
        it('should find the min data value', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getLowValue(data, 'x');

            expect(highX).toEqual(15);
        });

        it('should find the different min data value for a different key', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getLowValue(data, 'y');

            expect(highX).toEqual(25);
        });

        it('should find the next multiple lower than the min data value, when multiple supplied', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getLowValue(data, 'x', 10);

            expect(highX).toEqual(10);
        });

        it('should find the next multiple lower than the min data value, even for distant multiple', () => {
            const data = [
                { x: 89, y: 47 },
                { x: 15, y: 197 },
                { x: 23, y: 777 },
                { x: 65, y: 25 },
            ];

            const highX = getLowValue(data, 'y', 100);

            expect(highX).toEqual(0);
        });
    });
});
