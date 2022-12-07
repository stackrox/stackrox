import isEqual from 'lodash/isEqual';
import { useState, useCallback } from 'react';
import { CollectionRequest, CollectionResponse } from 'services/CollectionsService';
import { generateRequest } from '../converter';
import { Collection } from '../types';

export default function useDryRunConfiguration(
    collectionId: string | undefined,
    initialData: Omit<CollectionResponse, 'id'>
) {
    const id = collectionId;
    const [dryRunConfig, setDryRunConfig] = useState<CollectionRequest>(() => {
        // Use the `CollectionResponse` from the initial `getCollection` request here to support
        // collections that have configs unsupported from the UI, but can also have results displayed
        const { name, description, embeddedCollections, resourceSelectors } = initialData;
        const embeddedCollectionIds = embeddedCollections.map((collection) => collection.id);
        return { id, name, description, resourceSelectors, embeddedCollectionIds };
    });

    const updateDryRunConfig = useCallback(
        (values: Collection) => {
            const nextConfig = { id, ...generateRequest(values) };
            const isEmbeddedCollectionsChanged = !isEqual(
                dryRunConfig.embeddedCollectionIds,
                nextConfig.embeddedCollectionIds
            );
            const isResourceSelectorChanged = !isEqual(
                dryRunConfig.resourceSelectors,
                nextConfig.resourceSelectors
            );
            if (isEmbeddedCollectionsChanged || isResourceSelectorChanged) {
                setDryRunConfig(nextConfig);
            }
        },
        [dryRunConfig.embeddedCollectionIds, dryRunConfig.resourceSelectors, id]
    );

    return {
        dryRunConfig,
        updateDryRunConfig,
    };
}
