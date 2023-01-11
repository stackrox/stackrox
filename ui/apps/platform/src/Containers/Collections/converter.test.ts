import { CollectionRequest, Collection } from 'services/CollectionsService';
import { generateRequest, isCollectionParseError, parseCollection } from './converter';
import { ClientCollection } from './types';

describe('Collection parser', () => {
    it('should convert between BE CollectionResponse and FE Collection', () => {
        const collectionResponse: Collection = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'OR',
                            fieldName: 'Cluster',
                            values: [{ value: 'production', matchType: 'EXACT' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.name=backend',
                                    matchType: 'EXACT',
                                },
                                {
                                    value: 'kubernetes.io/metadata.name=frontend',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.release=stable',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                    ],
                },
            ],
            embeddedCollections: [{ id: '12' }, { id: '13' }, { id: '14' }],
        };
        const expectedCollection: ClientCollection = {
            name: 'Sample',
            description: 'Sample description',
            resourceSelector: {
                Deployment: { type: 'All' },
                Namespace: {
                    type: 'ByLabel',
                    field: 'Namespace Label',
                    rules: [
                        {
                            operator: 'OR',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.name=backend',
                                    matchType: 'EXACT',
                                },
                                {
                                    value: 'kubernetes.io/metadata.name=frontend',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                        {
                            operator: 'OR',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.release=stable',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                    ],
                },
                Cluster: {
                    type: 'ByName',
                    field: 'Cluster',
                    rule: { operator: 'OR', values: [{ value: 'production', matchType: 'EXACT' }] },
                },
            },
            embeddedCollectionIds: ['12', '13', '14'],
        };
        const parsedResponse = parseCollection(collectionResponse) as ClientCollection;
        expect(isCollectionParseError(parsedResponse)).toBeFalsy();
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
        const collectionResponse: Collection = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [{ rules: [] }, { rules: [] }],
            embeddedCollections: [],
        };
        expect(isCollectionParseError(parseCollection(collectionResponse))).toBeTruthy();
    });

    it('should error on rules for multiple fields for a single entity', () => {
        const collectionResponse: Collection = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'OR',
                            fieldName: 'Cluster',
                            values: [{ value: 'production', matchType: 'EXACT' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Cluster Label',
                            values: [{ value: 'key=value', matchType: 'EXACT' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(isCollectionParseError(parseCollection(collectionResponse))).toBeTruthy();
    });

    it('should error on conjunction rules', () => {
        const collectionResponse: Collection = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'AND',
                            fieldName: 'Cluster',
                            values: [{ value: 'production', matchType: 'EXACT' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(isCollectionParseError(parseCollection(collectionResponse))).toBeTruthy();
    });

    it('should error on rules against annotation field names', () => {
        const collectionResponse: Collection = {
            id: 'a-b-c',
            name: 'Sample',
            description: 'Sample description',
            resourceSelectors: [
                {
                    rules: [
                        {
                            operator: 'AND',
                            fieldName: 'Cluster Annotation',
                            values: [{ value: 'production', matchType: 'EXACT' }],
                        },
                    ],
                },
            ],
            embeddedCollections: [],
        };

        expect(isCollectionParseError(parseCollection(collectionResponse))).toBeTruthy();
    });
});

describe('Collection response generator', () => {
    it('should convert between FE Collection and BE CollectionRequest', () => {
        const collection: ClientCollection = {
            name: 'Sample',
            description: 'Sample description',
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
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.name=backend',
                                    matchType: 'EXACT',
                                },
                                {
                                    value: 'kubernetes.io/metadata.name=frontend',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                        {
                            operator: 'OR',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.release=stable',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                    ],
                },
                // "ByName" will create a single name rule
                Cluster: {
                    type: 'ByName',
                    field: 'Cluster',
                    rule: { operator: 'OR', values: [{ value: 'production', matchType: 'EXACT' }] },
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
                            values: [{ value: 'production', matchType: 'EXACT' }],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.name=backend',
                                    matchType: 'EXACT',
                                },
                                {
                                    value: 'kubernetes.io/metadata.name=frontend',
                                    matchType: 'EXACT',
                                },
                            ],
                        },
                        {
                            operator: 'OR',
                            fieldName: 'Namespace Label',
                            values: [
                                {
                                    value: 'kubernetes.io/metadata.release=stable',
                                    matchType: 'EXACT',
                                },
                            ],
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
