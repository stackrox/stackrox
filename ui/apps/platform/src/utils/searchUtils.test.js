import {
    getViewStateFromSearch,
    filterAllowedSearch,
    convertToRestSearch,
    convertSortToGraphQLFormat,
    convertSortToRestFormat,
    searchOptionsToSearchFilter,
    getListQueryParams,
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

    describe('filterAllowedSearch', () => {
        it('should return an empty object for an empty object', () => {
            const allowedOptions = [
                'Annotation',
                'Deployment',
                'Image',
                'Image Created Time',
                'Label',
                'Namespace',
                'Priority',
                'Secret',
                'Service Account',
            ];
            const pageSearch = {};

            const allowedSearch = filterAllowedSearch(allowedOptions, pageSearch);

            expect(allowedSearch).toEqual({});
        });

        it('should pass through all terms when allowed', () => {
            const allowedOptions = [
                'Annotation',
                'Deployment',
                'Image',
                'Image Created Time',
                'Label',
                'Namespace',
                'Priority',
                'Secret',
                'Service Account',
            ];
            const pageSearch = {
                Deployment: 'nginx',
                Label: 'web',
                Namespace: 'production',
            };

            const allowedSearch = filterAllowedSearch(allowedOptions, pageSearch);

            expect(allowedSearch).toEqual(pageSearch);
        });

        it('should filter out unallowed terms', () => {
            const allowedOptions = [
                'Annotation',
                'Deployment',
                'Image',
                'Image Created Time',
                'Label',
                'Namespace',
                'Priority',
                'Secret',
                'Service Account',
            ];
            const pageSearch = {
                Deployment: 'nginx',
                Label: 'web',
                Marco: 'polo',
                Namespace: 'production',
            };

            const allowedSearch = filterAllowedSearch(allowedOptions, pageSearch);

            expect(allowedSearch).toEqual({
                Deployment: 'nginx',
                Label: 'web',
                Namespace: 'production',
            });
        });
    });

    describe('convertToRestSearch', () => {
        it('should return an empty array when passed null', () => {
            const pageSearch = null;

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

    describe('searchOptionsToSearchFilter', () => {
        it('should translate an array of SearchEntries to a SearchFilter object', () => {
            expect(
                searchOptionsToSearchFilter([
                    { type: 'categoryOption', value: 'Image', label: 'Image' },
                    { value: 'nginx:latest', label: 'nginx:latest' },
                    { type: 'categoryOption', value: 'Status', label: 'Status' },
                    { type: 'categoryOption', value: 'Severity', label: 'Severity' },
                    { value: 'LOW_SEVERITY', label: 'LOW_SEVERITY' },
                    { value: 'HIGH_SEVERITY', label: 'HIGH_SEVERITY' },
                ])
            ).toEqual({
                Image: 'nginx:latest',
                Status: '',
                Severity: ['LOW_SEVERITY', 'HIGH_SEVERITY'],
            });
        });

        it('should return an empty string value when no search options is provided for a category', () => {
            expect(
                searchOptionsToSearchFilter([
                    { type: 'categoryOption', value: 'Status', label: 'Status' },
                ])
            ).toEqual({ Status: '' });
        });

        it('should return a string value when a single search options is provided for a category', () => {
            expect(
                searchOptionsToSearchFilter([
                    { type: 'categoryOption', value: 'Image', label: 'Image' },
                    { value: 'nginx:latest', label: 'nginx:latest' },
                ])
            ).toEqual({ Image: 'nginx:latest' });
        });

        it('should return an array value when multiple search options are provided for a category', () => {
            expect(
                searchOptionsToSearchFilter([
                    { type: 'categoryOption', value: 'Severity', label: 'Severity' },
                    { value: 'LOW_SEVERITY', label: 'LOW_SEVERITY' },
                    { value: 'HIGH_SEVERITY', label: 'HIGH_SEVERITY' },
                ])
            ).toEqual({ Severity: ['LOW_SEVERITY', 'HIGH_SEVERITY'] });
        });
    });

    describe('getListQueryParams', () => {
        it('should include all provided parameters in the query string', () => {
            expect(
                getListQueryParams(
                    { Deployment: ['visa-processor', 'scanner'] },
                    { field: 'Name', reversed: false },
                    0,
                    20
                )
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
            expect(getListQueryParams({}, { field: 'Name', reversed: false }, 0, 20)).toEqual(
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
            expect(getListQueryParams({}, { field: 'Name', reversed: false }, -1, 20)).toContain(
                'pagination.offset=0'
            );
            expect(getListQueryParams({}, { field: 'Name', reversed: false }, -100, 20)).toContain(
                'pagination.offset=0'
            );
            expect(
                getListQueryParams({}, { field: 'Name', reversed: false }, -Infinity, 20)
            ).toContain('pagination.offset=0');
        });

        it('should ensure that the offset is always a multiple of the page number', () => {
            const testValues = [
                [1, 10],
                [3, 10],
                [5, 5],
                [10, 3],
                [10, 1],
            ];

            testValues.forEach(([page, pageSize]) => {
                const params = getListQueryParams(
                    {},
                    { field: 'Name', reversed: false },
                    page,
                    pageSize
                );
                const [, offsetParam] = params.match(/pagination.offset=(\d+)/);
                expect(offsetParam).not.toBe('');
                const offset = parseInt(offsetParam, 10);
                expect(offset % page).toBe(0);
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
