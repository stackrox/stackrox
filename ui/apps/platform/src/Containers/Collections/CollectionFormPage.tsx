import React from 'react';
import { CollectionPageAction } from './collections.utils';

export type CollectionsFormPageProps = {
    pageAction: CollectionPageAction;
};

function CollectionsFormPage({ pageAction }: CollectionsFormPageProps) {
    return pageAction.type === 'create' ? (
        <>{pageAction.type}</>
    ) : (
        <>
            {pageAction.type} {pageAction.collectionId}
        </>
    );
}

export default CollectionsFormPage;
