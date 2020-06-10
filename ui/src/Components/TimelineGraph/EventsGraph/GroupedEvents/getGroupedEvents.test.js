import getGroupedEvents from './getGroupedEvents';

describe('getGroupedEvents', () => {
    it('should return grouped events', () => {
        const events = [
            { differenceInMilliseconds: 25 },
            { differenceInMilliseconds: 50 },
            { differenceInMilliseconds: 75 },
            { differenceInMilliseconds: 100 },
            { differenceInMilliseconds: 125 },
            { differenceInMilliseconds: 150 },
            { differenceInMilliseconds: 175 },
            { differenceInMilliseconds: 200 },
        ];
        const minDomain = 0;
        const maxDomain = 200;
        const minRange = 0;
        const maxRange = 200;
        const partitionSize = 100;

        const groupedEvents = getGroupedEvents({
            events,
            minDomain,
            maxDomain,
            minRange,
            maxRange,
            partitionSize,
        });

        expect(groupedEvents).toEqual([
            {
                differenceInMilliseconds: 0,
                events: [
                    { differenceInMilliseconds: 25 },
                    { differenceInMilliseconds: 50 },
                    { differenceInMilliseconds: 75 },
                ],
            },
            {
                differenceInMilliseconds: 200,
                events: [
                    { differenceInMilliseconds: 100 },
                    { differenceInMilliseconds: 125 },
                    { differenceInMilliseconds: 150 },
                    { differenceInMilliseconds: 175 },
                    { differenceInMilliseconds: 200 },
                ],
            },
        ]);
    });
});
