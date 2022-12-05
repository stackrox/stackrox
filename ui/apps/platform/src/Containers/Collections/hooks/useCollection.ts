import { useCallback } from 'react';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import {
    CollectionResponse,
    getCollection,
    listCollections,
    ResolvedCollectionResponseWithMatches,
} from 'services/CollectionsService';

const defaultCollectionData: Omit<CollectionResponse, 'id'> = {
    name: '',
    description: '',
    inUse: false,
    embeddedCollections: [],
    resourceSelectors: [],
};

const noopRequest = {
    request: Promise.resolve<{
        collection: Omit<CollectionResponse, 'id'>;
        embeddedCollections: CollectionResponse[];
    }>({ collection: defaultCollectionData, embeddedCollections: [] }),
    cancel: () => {},
};

function getEmbeddedCollections({ collection }: ResolvedCollectionResponseWithMatches): Promise<{
    collection: CollectionResponse;
    embeddedCollections: CollectionResponse[];
}> {
    if (collection.embeddedCollections.length === 0) {
        return Promise.resolve({ collection, embeddedCollections: [] });
    }
    const idSearchString = collection.embeddedCollections.map(({ id }) => id).join(',');
    const searchFilter = { 'Collection ID': idSearchString };
    const { request } = listCollections(searchFilter, {
        field: 'Collection Name',
        reversed: false,
    });
    return request.then((embeddedCollections) => ({ collection, embeddedCollections }));
}

export default function useCollection(collectionId: string | undefined) {
    const collectionFetcher = useCallback(() => {
        if (!collectionId) {
            return noopRequest;
        }
        const { request, cancel } = getCollection(collectionId);
        return { request: request.then(getEmbeddedCollections), cancel };
    }, [collectionId]);

    return useRestQuery(collectionFetcher);
}
