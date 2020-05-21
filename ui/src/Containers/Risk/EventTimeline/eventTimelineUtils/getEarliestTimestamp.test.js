import getEarliestTimestamp from './getEarliestTimestamp';

describe('getEarliestTimestamp', () => {
    it('should return the earliest timestamp from multiple timestamps', () => {
        const timestamps = [
            '2020-04-20T16:00:00Z',
            '2020-04-21T16:00:00Z',
            '2020-04-22T16:00:00Z',
            '2020-04-23T16:00:00Z',
            '2020-04-24T16:00:00Z',
        ];

        const earliestTimestamp = getEarliestTimestamp(timestamps);

        expect(earliestTimestamp).toEqual('2020-04-20T16:00:00Z');
    });

    it('should return the only timestamp provided', () => {
        const timestamps = ['2020-04-20T16:00:00Z'];

        const earliestTimestamp = getEarliestTimestamp(timestamps);

        expect(earliestTimestamp).toEqual('2020-04-20T16:00:00Z');
    });

    it('should return null when there are no timestamps', () => {
        const timestamps = [];

        const earliestTimestamp = getEarliestTimestamp(timestamps);

        expect(earliestTimestamp).toEqual(null);
    });

    it('should return null when null is provided', () => {
        const timestamps = null;

        const earliestTimestamp = getEarliestTimestamp(timestamps);

        expect(earliestTimestamp).toEqual(null);
    });
});
