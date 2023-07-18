import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import {
    Collection,
    getCollection,
    listCollections,
    CollectionResponseWithMatches,
} from 'services/CollectionsService';

const defaultCollectionData: Omit<Collection, 'id'> = {
    name: '',
    description: '',
    embeddedCollections: [],
    resourceSelectors: [],
};

const noopRequest = {
    request: Promise.resolve<{
        collection: Omit<Collection, 'id'>;
        embeddedCollections: Collection[];
    }>({ collection: defaultCollectionData, embeddedCollections: [] }),
    cancel: () => {},
};

function getEmbeddedCollections({ collection }: CollectionResponseWithMatches): Promise<{
    collection: Collection;
    embeddedCollections: Collection[];
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
