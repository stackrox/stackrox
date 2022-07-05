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

        it('correctly quote wraps exact search term values', () => {
            const singleValueQueryObject = {
                'Cluster ID': '8afe70aa-0092-4c3f-a744-d259e863e262',
            };

            const singleValueQueryString = queryService.objectToWhereClause(singleValueQueryObject);

            expect(singleValueQueryString).toEqual(
                'Cluster ID:"8afe70aa-0092-4c3f-a744-d259e863e262"'
            );

            const multiValueQueryObject = {
                'Cluster ID': [
                    '8afe70aa-0092-4c3f-a744-d259e863e262',
                    '7c9786b3-d436-40f2-aff5-6ea758d15830',
                    '395b243e-bd99-404b-96d1-493768f54ca3',
                ],
            };

            const multiValueQueryString = queryService.objectToWhereClause(multiValueQueryObject);

            expect(multiValueQueryString).toEqual(
                'Cluster ID:"8afe70aa-0092-4c3f-a744-d259e863e262","7c9786b3-d436-40f2-aff5-6ea758d15830","395b243e-bd99-404b-96d1-493768f54ca3"'
            );
        });
    });

    describe('entityContextToQueryObject', () => {
        it('returns the query object for a Component', () => {
            const entityContext = {
                COMPONENT: 'cHl0aG9uMy40:My40LjMtMXVidW50dTF-MTQuMDQuNw',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                COMPONENT: 'python3.4',
                'COMPONENT VERSION': '3.4.3-1ubuntu1~14.04.7',
            });
        });
    });
});
