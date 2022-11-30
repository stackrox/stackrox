import { useState } from 'react';

import {
    updateCollection,
    createCollection,
    CollectionResponse,
} from 'services/CollectionsService';
import { CollectionPageAction } from '../collections.utils';
import { generateRequest } from '../converter';
import { CollectionSaveError, parseSaveError } from '../errorUtils';
import { Collection } from '../types';

export function useCollectionFormSubmission(pageAction: CollectionPageAction) {
    const [saveError, setSaveError] = useState<CollectionSaveError | undefined>();

    function onSubmit(collection: Collection): Promise<CollectionResponse> {
        setSaveError(undefined);

        return new Promise<CollectionResponse>((resolve, reject) => {
            if (pageAction.type === 'view') {
                // Logically should not happen, but just in case
                return reject(new Error('A Collection form has been submitted in read-only view'));
            }
            const isEmptyCollection =
                Object.values(collection.resourceSelector).every(({ type }) => type === 'All') &&
                collection.embeddedCollectionIds.length === 0;

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
            setSaveError(parseSaveError(err));
            return Promise.reject(err);
        });
    }

    return {
        saveError,
        setSaveError,
        onSubmit,
    };
}
