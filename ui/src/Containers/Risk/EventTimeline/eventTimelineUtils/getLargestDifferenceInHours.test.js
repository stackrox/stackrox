import getLargestDifferenceInHours from './getLargestDifferenceInHours';

describe('getLargestDifferenceInHours', () => {
    it('should return the largest difference in hours', () => {
        const timelineData = [
            {
                events: [{ differenceInHours: 3 }, {}, { differenceInHours: 1 }]
            },
            {
                events: []
            },
            {
                events: [
                    { differenceInHours: 0 },
                    { differenceInHours: 10 },
                    { differenceInHours: 1 }
                ]
            }
        ];

        const value = getLargestDifferenceInHours(timelineData);

        expect(value).toEqual(10);
    });

    it('should return 0 with no data', () => {
        const timelineData = [];

        const value = getLargestDifferenceInHours(timelineData);

        expect(value).toEqual(0);
    });
});
