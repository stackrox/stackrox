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

    describe('entityContextToQueryObject', () => {
        it('returns the query object for a Component', () => {
            const entityContext = {
                COMPONENT: 'cHl0aG9uMy40:My40LjMtMXVidW50dTF-MTQuMDQuNw',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'COMPONENT NAME': 'python3.4',
                'COMPONENT VERSION': '3.4.3-1ubuntu1~14.04.7',
            });
        });
    });
});
