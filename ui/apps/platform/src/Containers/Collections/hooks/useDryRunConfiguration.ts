import isEqual from 'lodash/isEqual';
import { useState, useCallback } from 'react';
import { Collection, CollectionRequest } from 'services/CollectionsService';
import { generateRequest } from '../converter';
import { ClientCollection } from '../types';

export default function useDryRunConfiguration(
    collectionId: string | undefined,
    initialData: Omit<Collection, 'id'>
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
        (values: ClientCollection) => {
            const nextConfig = { id, ...generateRequest(values) };
            const isEmbeddedCollectionsChanged = !isEqual(
                dryRunConfig.embeddedCollectionIds,
                nextConfig.embeddedCollectionIds
            );
            const isResourceSelectorChanged = !isEqual(
                dryRunConfig.resourceSelectors,
                nextConfig.resourceSelectors
            );
            const isNameChange = dryRunConfig.name !== nextConfig.name;
            if (isEmbeddedCollectionsChanged || isResourceSelectorChanged || isNameChange) {
                setDryRunConfig(nextConfig);
            }
        },
        [dryRunConfig.embeddedCollectionIds, dryRunConfig.name, dryRunConfig.resourceSelectors, id]
    );

    // This function forces an update of the configuration in order to trigger a refetch
    // of the collection results.
    // TODO - As a follow up, investigate pulling the state and fetching code from `CollectionResults` instead.
    const refreshConfig = useCallback(() => setDryRunConfig((config) => ({ ...config })), []);

    return {
        dryRunConfig,
        updateDryRunConfig,
        refreshConfig,
    };
}
