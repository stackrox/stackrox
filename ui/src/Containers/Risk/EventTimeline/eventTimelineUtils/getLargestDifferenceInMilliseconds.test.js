import getLargestDifferenceInMilliseconds from './getLargestDifferenceInMilliseconds';

describe('getLargestDifferenceInMilliseconds', () => {
    it('should return the largest difference in milliseconds', () => {
        const timelineData = [
            {
                events: [
                    { differenceInMilliseconds: 10000 },
                    {},
                    { differenceInMilliseconds: 10000 }
                ]
            },
            {
                events: []
            },
            {
                events: [
                    { differenceInMilliseconds: 100 },
                    { differenceInMilliseconds: 100000 },
                    { differenceInMilliseconds: 5000 }
                ]
            }
        ];

        const value = getLargestDifferenceInMilliseconds(timelineData);

        expect(value).toEqual(100000);
    });

    it('should return 0 with no data', () => {
        const timelineData = [];

        const value = getLargestDifferenceInMilliseconds(timelineData);

        expect(value).toEqual(0);
    });
});
