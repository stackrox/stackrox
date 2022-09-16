// eslint-disable-next-line import/no-extraneous-dependencies
import { rest } from 'msw';
import sortBy from 'lodash/sortBy';

import {
    CollectionRequest,
    CollectionResponse,
    collectionsAutocompleteUrl,
    collectionsBaseUrl,
    collectionsCountUrl,
    collectionsDryRunUrl,
} from 'services/CollectionsService';

const collectionsStore: CollectionResponse[] = [
    {
        id: '1',
        name: 'Notable deployments',
        description:
            'A group of deployments that are unique or otherwise interesting in a way that we would want to group them in a collection. It is important that this collection also makes it clear that the description should be long.',
        inUse: true,
        resourceSelectors: [
            {
                rules: [
                    {
                        fieldName: 'Namespace',
                        operator: 'OR',
                        values: [{ value: 'stackrox' }],
                    },
                ],
            },
        ],
        embeddedCollections: [],
    },
    {
        id: '2',
        name: 'Collection++',
        description: 'A more complicated and misunderstood collection',
        inUse: false,
        resourceSelectors: [
            {
                rules: [
                    {
                        fieldName: 'Namespace',
                        operator: 'OR',
                        values: [{ value: 'stackrox' }, { value: 'kube-system' }],
                    },
                ],
            },
        ],
        embeddedCollections: [{ id: '1' }],
    },
    {
        id: '3',
        name: 'Payment processing team',
        description: 'deployments belonging to the payments team',
        inUse: true,
        resourceSelectors: [
            {
                rules: [
                    {
                        fieldName: 'Cluster',
                        operator: 'OR',
                        values: [{ value: 'production' }],
                    },
                    {
                        fieldName: 'Deployment Label',
                        operator: 'OR',
                        values: [{ value: 'payments' }],
                    },
                ],
            },
        ],
        embeddedCollections: [],
    },
];

const CollectionsService = [
    // listCollections
    rest.get<{
        collections: CollectionResponse[];
    }>(collectionsBaseUrl, (req, res, ctx) => {
        const params = req.url.searchParams;
        const query = params.get('query')?.replace(/Collection:/, '') ?? '';
        const offset = Number(params.get('pagination.offset'));
        const limit = Number(params.get('pagination.limit'));
        const field = params.get('pagination.sortOption.field') ?? '';
        const reversed = params.get('pagination.sortOption.reversed') === 'true';

        const sorted = sortBy(
            collectionsStore.filter((c) => c.name.toLowerCase().includes(query.toLowerCase())),
            field
        );
        if (reversed) {
            sorted.reverse();
        }

        const collections = sorted.slice(offset, limit + offset);
        return res(ctx.json({ collections }));
    }),
    // getCollectionCount
    rest.get<{ count: string }>(collectionsCountUrl, (req, res, ctx) => {
        const params = req.url.searchParams;
        const query = params.get('query')?.replace(/Collection:/, '') ?? '';
        const collections = collectionsStore.filter((c) =>
            c.name.toLowerCase().includes(query.toLowerCase())
        );
        return res(
            ctx.json({
                count: collections.length,
            })
        );
    }),
    // getCollection
    rest.get(`${collectionsBaseUrl}/:id`, (req, res, ctx) => {
        const target = collectionsStore.find((c) => c.id === req.params.id);
        return target ? res(ctx.json(target)) : res(ctx.status(404));
    }),
    // createCollection
    rest.post<CollectionRequest>(collectionsBaseUrl, async (req, res, ctx) => {
        const { name, description, resourceSelectors, embeddedCollectionIds }: CollectionRequest =
            await req.json();
        const newCollection = {
            id: `${collectionsStore.length + 1}`,
            name,
            description,
            resourceSelectors,
            inUse: false,
            embeddedCollections: embeddedCollectionIds.map((id) => ({ id })),
        };
        collectionsStore.push(newCollection);
        return res(ctx.json(newCollection));
    }),
    // updateCollection
    rest.post(`${collectionsBaseUrl}/:id`, async (req, res, ctx) => {
        const target = collectionsStore.find((c) => c.id === req.params.id);
        if (!target) {
            return res(ctx.status(404));
        }

        const { name, description, resourceSelectors, embeddedCollectionIds }: CollectionRequest =
            await req.json();
        target.name = name;
        target.description = description;
        target.resourceSelectors = resourceSelectors;
        target.embeddedCollections = embeddedCollectionIds.map((id) => ({ id }));
        return res(ctx.json(target));
    }),
    // deleteCollection
    rest.delete(`${collectionsBaseUrl}/:id`, (req, res, ctx) => {
        const target = collectionsStore.find((c) => c.id === req.params.id);
        if (!target) {
            return res(ctx.status(404));
        }

        collectionsStore.splice(collectionsStore.indexOf(target), 1);
        return res(ctx.json({}));
    }),
    // dryRunCollection
    rest.post(collectionsDryRunUrl, (req, res, ctx) => {
        return res(ctx.status(404, 'This MSW request has been intercepted but not implemented'));
    }),
    // getCollectionAutoComplete
    rest.get(collectionsAutocompleteUrl, (req, res, ctx) => {
        return res(ctx.status(404, 'This MSW request has been intercepted but not implemented'));
    }),
];

export default CollectionsService;
