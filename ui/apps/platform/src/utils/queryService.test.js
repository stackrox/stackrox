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
        it('returns the query object for an Image Component', () => {
            const entityContext = {
                IMAGE_COMPONENT: 'openssl#1.1.1d-0+deb10u7#debian:10',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'COMPONENT ID': 'openssl#1.1.1d-0+deb10u7#debian:10',
            });
        });

        it('returns the query object for a Node Component', () => {
            const entityContext = {
                NODE_COMPONENT: 'linux-gke#5.4.0-1068#ubuntu:20.04',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'COMPONENT ID': 'linux-gke#5.4.0-1068#ubuntu:20.04',
            });
        });

        it('returns the query object for an IMAGE CVE', () => {
            const entityContext = {
                IMAGE_CVE: 'CVE-2005-2541#debian:10',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'CVE ID': 'CVE-2005-2541#debian:10',
            });
        });

        it('returns the query object for a NODE CVE', () => {
            const entityContext = {
                NODE_CVE: 'CVE-2022-27223#ubuntu:20.04',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'CVE ID': 'CVE-2022-27223#ubuntu:20.04',
            });
        });

        it('returns the query object for a CLUSTER CVE', () => {
            const entityContext = {
                CLUSTER_CVE: 'CVE-2020-8554#K8S_CVE',
            };

            const queryObject = queryService.entityContextToQueryObject(entityContext);

            expect(queryObject).toEqual({
                'CVE ID': 'CVE-2020-8554#K8S_CVE',
            });
        });
    });
});
