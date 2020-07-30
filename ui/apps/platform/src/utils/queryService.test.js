import queryService from 'utils/queryService';

describe('queryService.', () => {
    describe('objectToWhereClause', () => {
        it('returns an empty string when passed an empty object', () => {
            const queryObj = {};

            const queryStr = queryService.objectToWhereClause(queryObj);

            expect(queryStr).toEqual('');
        });

        it('converts an option to a GraphQL query string', () => {
            const cvesStr = 'CVE-2005-2541,CVE-2017-12424,CVE-2018-16402';
            const queryObj = {
                cve: cvesStr,
            };

            const queryStr = queryService.objectToWhereClause(queryObj);

            expect(queryStr).toEqual('cve:CVE-2005-2541,CVE-2017-12424,CVE-2018-16402');
        });
    });
});
