import { CollectionRequest, CollectionResponse } from 'services/CollectionsService';
import { generateRequest, parseCollection } from './converter';
import { Collection } from './types';

describe('Collection parser', () => {
    it('should convert between BE CollectionResponse and FE Collection', () => {
        const collectionResponse: CollectionResponse = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'OR',
                            fieldName: 'Cluster',
                            values: [{ value: 'production' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                { value: 'kubernetes.io/metadata.name=backend' },
                                { value: 'kubernetes.io/metadata.name=frontend' },
                            ],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [{ value: 'kubernetes.io/metadata.release=stable' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [{ id: '12' }, { id: '13' }, { id: '14' }],
        };
        const expectedCollection: Collection = {
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelector: {
                Deployment: { type: 'All' },
                Namespace: {
                    type: 'ByLabel',
                    field: 'Namespace Label',
                    rules: [
                        {
                            operator: 'OR',
                            key: 'kubernetes.io/metadata.name',
                            values: ['backend', 'frontend'],
                        },
                        {
                            operator: 'OR',
                            key: 'kubernetes.io/metadata.release',
                            values: ['stable'],
                        },
                    ],
                },
                Cluster: {
                    type: 'ByName',
                    field: 'Cluster',
                    rule: { operator: 'OR', values: ['production'] },
                },
            },
            embeddedCollectionIds: ['12', '13', '14'],
        };
        const parsedResponse = parseCollection(collectionResponse) as Collection;
        expect(parsedResponse).not.toBeInstanceOf(AggregateError);
        expect(parsedResponse.id).toEqual(expectedCollection.id);
        expect(parsedResponse.name).toEqual(expectedCollection.name);
        expect(parsedResponse.description).toEqual(expectedCollection.description);
        expect(parsedResponse.embeddedCollectionIds).toEqual(
            expect.arrayContaining(expectedCollection.embeddedCollectionIds)
        );
        expect(parsedResponse.resourceSelector.Cluster).toEqual(
            expectedCollection.resourceSelector.Cluster
        );
    });

    it('should error on multiple resource selectors', () => {
        const collectionResponse: CollectionResponse = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelectors: [{ rules: [] }, { rules: [] }],
            embeddedCollections: [],
        };
        expect(parseCollection(collectionResponse)).toBeInstanceOf(AggregateError);
    });

    it('should error on rules for multiple fields for a single entity', () => {
        const collectionResponse: CollectionResponse = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'OR',
                            fieldName: 'Cluster',
                            values: [{ value: 'production' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Cluster Label',
                            values: [{ value: 'key=value' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(parseCollection(collectionResponse)).toBeInstanceOf(AggregateError);
    });

    it('should error on conjunction rules', () => {
        const collectionResponse: CollectionResponse = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'AND',
                            fieldName: 'Cluster',
                            values: [{ value: 'production' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(parseCollection(collectionResponse)).toBeInstanceOf(AggregateError);
    });

    it('should error on rules against annotation field names', () => {
        const collectionResponse: CollectionResponse = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'AND',
                            fieldName: 'Cluster Annotation',
                            values: [{ value: 'production' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(parseCollection(collectionResponse)).toBeInstanceOf(AggregateError);
    });
});

describe('Collection response generator', () => {
    it('should convert between FE Collection and BE CollectionRequest', () => {
        const collection: Collection = {
            name: 'Sample',
            description: 'Sample description',
            inUse: false,
            resourceSelector: {
                // "All" should result in no rules
                Deployment: { type: 'All' },
                // "ByLabel" will create two rules, one with multiple values, and test the joining of keys'values
                Namespace: {
                    type: 'ByLabel',
                    field: 'Namespace Label',
                    rules: [
                        {
                            operator: 'OR',
                            key: 'kubernetes.io/metadata.name',
                            values: ['backend', 'frontend'],
                        },
                        {
                            operator: 'OR',
                            key: 'kubernetes.io/metadata.release',
                            values: ['stable'],
                        },
                    ],
                },
                // "ByName" will create a single name rule
                Cluster: {
                    type: 'ByName',
                    field: 'Cluster',
                    rule: { operator: 'OR', values: ['production'] },
                },
            },
            embeddedCollectionIds: ['12', '13', '14'],
        };

        const expectedRequest: CollectionRequest = {
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'OR',
                            fieldName: 'Cluster',
                            values: [{ value: 'production' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                { value: 'kubernetes.io/metadata.name=backend' },
                                { value: 'kubernetes.io/metadata.name=frontend' },
                            ],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [{ value: 'kubernetes.io/metadata.release=stable' }],
                        },
                    ],
                },
            ],
            embeddedCollectionIds: ['12', '13', '14'],
        };

        const generatedRequest = generateRequest(collection);
        expect(generatedRequest.name).toEqual(expectedRequest.name);
        expect(generatedRequest.description).toEqual(expectedRequest.description);
        expect(generatedRequest.embeddedCollectionIds).toEqual(
            expect.arrayContaining(expectedRequest.embeddedCollectionIds)
        );
        expect(generatedRequest.resourceSelectors).toEqual(
            expect.arrayContaining(expectedRequest.resourceSelectors)
        );
    });
});
