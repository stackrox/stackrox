import searchOptionsToQuery from './searchOptionsToQuery';

describe('searchOptionsToQuery', () => {
    it('should return empty string for 0 options', () => {
        const options = [];
        const expected = '';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });

    it('should return query string for 2 options which do not have type', () => {
        const options = [
            { type: 'categoryOption', value: 'Orchestrator Component:' },
            { value: 'false' },
        ];
        const expected = 'Orchestrator Component:false';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });

    it('should return query string for 2 options which do have type and value string', () => {
        const options = [{ type: 'categoryOption', value: 'Cluster:' }, { value: 'remote' }];
        const expected = 'Cluster:remote';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });

    it('should return query string for 2 options which do have type and value array', () => {
        const options = [
            { type: 'categoryOption', value: 'CVE:' },
            { value: ['CVE-2021-38200', 'CVE-2021-38201'] },
        ];
        const expected = 'CVE:CVE-2021-38200,CVE-2021-38201';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });

    it('should return query string for 4 options which do have type', () => {
        const options = [
            { type: 'categoryOption', value: 'Cluster:' },
            { value: 'remote' },
            { type: 'categoryOption', value: 'Namespace:' },
            { value: 'stackrox' },
        ];
        const expected = 'Cluster:remote+Namespace:stackrox';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });

    it('should return query string for 4 options which do have type followed by 2 options which do not', () => {
        const options = [
            { type: 'categoryOption', value: 'Cluster:' },
            { value: 'remote' },
            { type: 'categoryOption', value: 'Namespace:' },
            { value: 'stackrox' },
            { type: 'categoryOption', value: 'Orchestrator Component:' },
            { value: 'false' },
        ];
        const expected = 'Cluster:remote+Namespace:stackrox+Orchestrator Component:false';
        expect(searchOptionsToQuery(options)).toEqual(expected);
    });
});
