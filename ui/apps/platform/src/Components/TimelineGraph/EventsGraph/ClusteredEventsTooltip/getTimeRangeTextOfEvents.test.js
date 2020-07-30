import getTimeRangeTextOfEvents from './getTimeRangeTextOfEvents';

describe('getTimeRangeTextOfEvents', () => {
    it('should show difference in milliseconds', () => {
        const events = [
            { differenceInMilliseconds: 25 },
            { differenceInMilliseconds: 50 },
            { differenceInMilliseconds: 200 },
        ];

        const groupedEvents = getTimeRangeTextOfEvents(events);

        expect(groupedEvents).toEqual('175 milliseconds');
    });

    it('should show difference in seconds', () => {
        const events = [
            { differenceInMilliseconds: 50 },
            { differenceInMilliseconds: 6000 },
            { differenceInMilliseconds: 300 },
        ];

        const groupedEvents = getTimeRangeTextOfEvents(events);

        expect(groupedEvents).toEqual('5 seconds');
    });

    it('should show difference in minutes and seconds', () => {
        const events = [
            { differenceInMilliseconds: 25000 },
            { differenceInMilliseconds: 80000 },
            { differenceInMilliseconds: 300 },
        ];

        const groupedEvents = getTimeRangeTextOfEvents(events);

        expect(groupedEvents).toEqual('1 minute and 19 seconds');
    });

    it('should show difference in hours and minutes', () => {
        const events = [
            { differenceInMilliseconds: 25000 },
            { differenceInMilliseconds: 80000 },
            { differenceInMilliseconds: 36000000 },
        ];

        const groupedEvents = getTimeRangeTextOfEvents(events);

        expect(groupedEvents).toEqual('9 hours and 59 minutes');
    });

    it('should show difference in days', () => {
        const events = [
            { differenceInMilliseconds: 25000 },
            { differenceInMilliseconds: 80000 },
            { differenceInMilliseconds: 360000000 },
        ];

        const groupedEvents = getTimeRangeTextOfEvents(events);

        expect(groupedEvents).toEqual('4 days');
    });
});
