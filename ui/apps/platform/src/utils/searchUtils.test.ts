import {
    getViewStateFromSearch,
    convertToRestSearch,
    convertSortToGraphQLFormat,
    convertSortToRestFormat,
    getListQueryParams,
    getPaginationParams,
    searchValueAsArray,
    convertToExactMatch,
} from './searchUtils';

describe('searchUtils', () => {
    describe('getViewStateFromSearch', () => {
        it('should return false when passed an empty search object', () => {
            const searchObj = {};
            const key = 'CVE Snoozed';

            const containsKey = getViewStateFromSearch(searchObj, key);

            expect(containsKey).toEqual(false);
        });

        it('should return false when key is not in the given search object', () => {
            const searchObj = { CVE: 'CVE-2019-9893' };
            const key = 'CVE Snoozed';

            const containsKey = getViewStateFromSearch(searchObj, key);

            expect(containsKey).toEqual(false);
        });

        it('should return true when key is in the given search object', () => {
            const searchObj = { 'CVE Snoozed': true, CVE: 'CVE-2019-9893' };
            const key = 'CVE Snoozed';

            const containsKey = getViewStateFromSearch(searchObj, key);

            expect(containsKey).toEqual(true);
        });

        it('should return false when key is in the given search object but its value is false', () => {
            const searchObj = { 'CVE Snoozed': 'false' };
            const key = 'CVE Snoozed';

            const containsKey = getViewStateFromSearch(searchObj, key);

            expect(containsKey).toEqual(false);
        });

        it('should return false when key is in the given search object but its value is string "false"', () => {
            const searchObj = { 'CVE Snoozed': false };
            const key = 'CVE Snoozed';

            const containsKey = getViewStateFromSearch(searchObj, key);

            expect(containsKey).toEqual(false);
        });
    });

    describe('convertToRestSearch', () => {
        it('should return an empty array when passed null', () => {
            const pageSearch = null;

            // @ts-expect-error Testing invalid input, the function signature does not accept null but it is
            //                  handled internally in the function
            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([]);
        });

        it('should return an empty array for an empty object', () => {
            const pageSearch = {};

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([]);
        });

        it('should return an array with 2 elements for an object with 1 key-value pair', () => {
            const pageSearch = { Namespace: 'docker' };

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([
                {
                    value: 'Namespace:',
                    label: 'Namespace:',
                    type: 'categoryOption',
                },
                {
                    value: 'docker',
                    label: 'docker',
                },
            ]);
        });

        it('should return an array with twice as many elements as an object has key-value pair', () => {
            const pageSearch = {
                Namespace: 'docker',
                Cluster: 'remote',
                Deployment: 'compose-api',
            };

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([
                {
                    value: 'Namespace:',
                    label: 'Namespace:',
                    type: 'categoryOption',
                },
                {
                    value: 'docker',
                    label: 'docker',
                },
                {
                    value: 'Cluster:',
                    label: 'Cluster:',
                    type: 'categoryOption',
                },
                {
                    value: 'remote',
                    label: 'remote',
                },
                {
                    value: 'Deployment:',
                    label: 'Deployment:',
                    type: 'categoryOption',
                },
                {
                    value: 'compose-api',
                    label: 'compose-api',
                },
            ]);
        });

        it('should not return a pair of array elements for object key without a value', () => {
            const pageSearch = { Namespace: '' };

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([]);
        });

        it('should return element pairs for complete key/value pairs, even it it does not return a pair of array elements for object key without a value', () => {
            const pageSearch = { Cluster: 'security', Namespace: '' };

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([
                {
                    value: 'Cluster:',
                    label: 'Cluster:',
                    type: 'categoryOption',
                },
                {
                    value: 'security',
                    label: 'security',
                },
            ]);
        });
    });

    describe('convertSortToGraphQLFormat', () => {
        it('should return an object the keys of the other object converted', () => {
            const restSort = {
                field: 'Priority',
                reversed: true,
            };

            const graphQLSort = convertSortToGraphQLFormat(restSort);

            expect(graphQLSort).toEqual({
                id: 'Priority',
                desc: true,
            });
        });
    });

    describe('convertSortToRestFormat', () => {
        it('should return an object the keys of the other object converted', () => {
            const restSort = [
                {
                    id: 'Priority',
                    desc: true,
                },
            ];

            const graphQLSort = convertSortToRestFormat(restSort);

            expect(graphQLSort).toEqual({
                field: 'Priority',
                reversed: true,
            });
        });
    });

    describe('getListQueryParams', () => {
        it('should include all provided parameters in the query string', () => {
            expect(
                getListQueryParams({
                    searchFilter: { Deployment: ['visa-processor', 'scanner'] },
                    sortOption: { field: 'Name', reversed: false },
                    page: 0,
                    perPage: 20,
                })
            ).toEqual(
                [
                    'query=Deployment%3Avisa-processor%2Cscanner',
                    'pagination.offset=0',
                    'pagination.limit=20',
                    'pagination.sortOption.field=Name',
                    'pagination.sortOption.reversed=false',
                ].join('&')
            );
        });

        it('should include pagination parameters when the search filter is empty', () => {
            expect(
                getListQueryParams({
                    searchFilter: {},
                    sortOption: { field: 'Name', reversed: false },
                    page: 0,
                    perPage: 20,
                })
            ).toEqual(
                [
                    'query=',
                    'pagination.offset=0',
                    'pagination.limit=20',
                    'pagination.sortOption.field=Name',
                    'pagination.sortOption.reversed=false',
                ].join('&')
            );
        });

        it('should ensure that negative pages result in an offset of 0', () => {
            expect(
                getListQueryParams({
                    searchFilter: {},
                    sortOption: { field: 'Name', reversed: false },
                    page: -1,
                    perPage: 20,
                })
            ).toContain('pagination.offset=0');
            expect(
                getListQueryParams({
                    searchFilter: {},
                    sortOption: { field: 'Name', reversed: false },
                    page: -100,
                    perPage: 20,
                })
            ).toContain('pagination.offset=0');
            expect(
                getListQueryParams({
                    searchFilter: {},
                    sortOption: { field: 'Name', reversed: false },
                    page: -Infinity,
                    perPage: 20,
                })
            ).toContain('pagination.offset=0');
        });

        it('should ensure that the offset is always a multiple of the page size', () => {
            const testValues = [
                [1, 10],
                [3, 10],
                [5, 5],
                [10, 3],
                [10, 1],
            ];

            testValues.forEach(([page, perPage]) => {
                const params = getListQueryParams({
                    searchFilter: {},
                    sortOption: { field: 'Name', reversed: false },
                    page,
                    perPage,
                });
                const matchArr = params.match(/pagination.offset=(\d+)/);
                const offsetParam = matchArr?.[1];
                expect(offsetParam).not.toBe('');
                expect(typeof offsetParam).toBe('string');
                const offset = parseInt(offsetParam as string, 10);
                expect(offset % perPage).toBe(0);
            });
        });
    });

    describe('getPaginationParams', () => {
        it('should calculate the offset based on the page number and page size', () => {
            expect(getPaginationParams({ page: 1, perPage: 20 })).toEqual({ offset: 0, limit: 20 });
            expect(getPaginationParams({ page: 2, perPage: 20 })).toEqual({
                offset: 20,
                limit: 20,
            });
            expect(getPaginationParams({ page: 3, perPage: 20 })).toEqual({
                offset: 40,
                limit: 20,
            });
            expect(getPaginationParams({ page: 4, perPage: 12 })).toEqual({
                offset: 36,
                limit: 12,
            });
            expect(getPaginationParams({ page: 5, perPage: 1 })).toEqual({ offset: 4, limit: 1 });
        });

        it('should include the optional sortOption parameter only when provided', () => {
            expect(
                getPaginationParams({
                    page: 1,
                    perPage: 20,
                    sortOption: { field: 'Name', reversed: false },
                })
            ).toEqual({
                offset: 0,
                limit: 20,
                sortOption: { field: 'Name', reversed: false },
            });

            expect(getPaginationParams({ page: 1, perPage: 20 })).toEqual({
                offset: 0,
                limit: 20,
            });
        });
    });

    describe('searchValueAsArray', () => {
        it('converts an undefined value to an empty array', () => {
            expect(searchValueAsArray(undefined)).toEqual([]);
        });

        it('converts a string value to an array with the string as the only element', () => {
            expect(searchValueAsArray('cluster')).toEqual(['cluster']);
        });

        it('converts an array value to an array with the same elements', () => {
            expect(searchValueAsArray([])).toEqual([]);
            expect(searchValueAsArray(['cluster', 'namespace'])).toEqual(['cluster', 'namespace']);
        });
    });

    describe('convertToExactMatch', () => {
        it('returns a non-string value unmodified', () => {
            expect(convertToExactMatch(undefined)).toEqual(undefined);
            expect(convertToExactMatch(null)).toEqual(null);
            expect(convertToExactMatch(42)).toEqual(42);
            expect(convertToExactMatch(['42'])).toEqual(['42']);
            expect(convertToExactMatch({ key: 'value' })).toEqual({ key: 'value' });
        });

        it('returns a string value wrapped in bespoke regex for exact match', () => {
            expect(convertToExactMatch('cluster')).toEqual('r/^cluster$');
        });
    });
});
