import getAggregateText from './getAggregateText';

describe('getAggregateText', () => {
    it('should return the unique value between all leaf values', () => {
        const leafValues = ['stackrox', 'stackrox', 'stackrox'];
        expect(getAggregateText(leafValues)).toEqual('stackrox');
    });

    it('should return the value "Many" when non-unique values are given', () => {
        const leafValues = ['stackrox', 'red hat', 'acs'];
        expect(getAggregateText(leafValues)).toEqual('Many');
    });

    it('should return the specified value when non-unique values are given', () => {
        const leafValues = ['stackrox', 'red hat', 'acs'];
        expect(getAggregateText(leafValues, 'Multiples')).toEqual('Multiples');
    });
});
