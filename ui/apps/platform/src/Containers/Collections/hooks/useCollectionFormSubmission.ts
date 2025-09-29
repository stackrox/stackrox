import { useState } from 'react';

import { updateCollection, createCollection } from 'services/CollectionsService';
import type { Collection } from 'services/CollectionsService';
import type { CollectionPageAction } from '../collections.utils';
import { generateRequest } from '../converter';
import { parseConfigError } from '../errorUtils';
import type { CollectionConfigError } from '../errorUtils';
import type { ClientCollection } from '../types';

export function useCollectionFormSubmission(pageAction: CollectionPageAction) {
    const [configError, setConfigError] = useState<CollectionConfigError | undefined>();

    function onSubmit(collection: ClientCollection): Promise<Collection> {
        setConfigError(undefined);

        return new Promise<Collection>((resolve, reject) => {
            if (pageAction.type === 'view') {
                // Logically should not happen, but just in case
                return reject(new Error('A Collection form has been submitted in read-only view'));
            }
            const isEmptyCollection =
                Object.values(collection.resourceSelector).every(
                    ({ type }) => type === 'NoneSpecified'
                ) && collection.embeddedCollectionIds.length === 0;

            if (isEmptyCollection) {
                return reject(new Error('Cannot save an empty collection'));
            }

            const saveServiceCall =
                pageAction.type === 'edit'
                    ? (payload) => updateCollection(pageAction.collectionId, payload)
                    : (payload) => createCollection(payload);

            const requestPayload = generateRequest(collection);
            const { request } = saveServiceCall(requestPayload);

            return resolve(request);
        }).catch((err) => {
            setConfigError(parseConfigError(err));
            return Promise.reject(err);
        });
    }

    return {
        configError,
        setConfigError,
        onSubmit,
    };
}
