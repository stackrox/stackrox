import {
    filterAllowedSearch,
    convertToRestSearch,
    convertSortToGraphQLFormat,
    convertSortToRestFormat
} from './riskPageUtils';

describe('riskPageUtils', () => {
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
                'Service Account'
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
                'Service Account'
            ];
            const pageSearch = {
                Deployment: 'nginx',
                Label: 'web',
                Namespace: 'production'
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
                'Service Account'
            ];
            const pageSearch = {
                Deployment: 'nginx',
                Label: 'web',
                Marco: 'polo',
                Namespace: 'production'
            };

            const allowedSearch = filterAllowedSearch(allowedOptions, pageSearch);

            expect(allowedSearch).toEqual({
                Deployment: 'nginx',
                Label: 'web',
                Namespace: 'production'
            });
        });
    });

    describe('convertToRestSearch', () => {
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
                    type: 'categoryOption'
                },
                {
                    value: 'docker',
                    label: 'docker'
                }
            ]);
        });

        it('should return an array with twice as many elements as an object has key-value pair', () => {
            const pageSearch = {
                Namespace: 'docker',
                Cluster: 'remote',
                Deployment: 'compose-api'
            };

            const restSearch = convertToRestSearch(pageSearch);

            expect(restSearch).toEqual([
                {
                    value: 'Namespace:',
                    label: 'Namespace:',
                    type: 'categoryOption'
                },
                {
                    value: 'docker',
                    label: 'docker'
                },
                {
                    value: 'Cluster:',
                    label: 'Cluster:',
                    type: 'categoryOption'
                },
                {
                    value: 'remote',
                    label: 'remote'
                },
                {
                    value: 'Deployment:',
                    label: 'Deployment:',
                    type: 'categoryOption'
                },
                {
                    value: 'compose-api',
                    label: 'compose-api'
                }
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
                    type: 'categoryOption'
                },
                {
                    value: 'security',
                    label: 'security'
                }
            ]);
        });
    });

    describe('convertSortToGraphQLFormat', () => {
        it('should return an object the keys of the other object converted', () => {
            const restSort = {
                field: 'Priority',
                reversed: true
            };

            const graphQLSort = convertSortToGraphQLFormat(restSort);

            expect(graphQLSort).toEqual({
                id: 'Priority',
                desc: true
            });
        });
    });

    describe('convertSortToRestFormat', () => {
        it('should return an object the keys of the other object converted', () => {
            const restSort = [
                {
                    id: 'Priority',
                    desc: true
                }
            ];

            const graphQLSort = convertSortToRestFormat(restSort);

            expect(graphQLSort).toEqual({
                field: 'Priority',
                reversed: true
            });
        });
    });
});
