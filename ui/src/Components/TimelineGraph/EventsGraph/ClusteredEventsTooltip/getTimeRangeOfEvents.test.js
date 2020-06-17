import getTimeRangeOfEvents from './getTimeRangeOfEvents';

describe('getTimeRangeOfEvents', () => {
    it('should get the time range oof events', () => {
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

        const groupedEvents = getTimeRangeOfEvents(events);

        expect(groupedEvents).toEqual({ timeRangeOfEvents: 175, unit: 'ms' });
    });
});
