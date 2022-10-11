import React, { useCallback } from 'react';
import { PageSection } from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import CollectionForm from './CollectionForm';

export type CollectionsFormPageProps = {
    hasWriteAccessForCollections: boolean;
    pageAction: CollectionPageAction;
};

const noopRequest = {
    request: Promise.resolve(undefined),
    cancel: () => {},
};

function CollectionsFormPage({
    hasWriteAccessForCollections,
    pageAction,
}: CollectionsFormPageProps) {
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const action = pageAction.type;
    const collectionId = action !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(
        () => (collectionId ? getCollection(collectionId) : noopRequest),
        [collectionId]
    );
    const { data } = useRestQuery(collectionFetcher);

    return (
        <>
            <PageSection className="pf-u-h-100" padding={{ default: 'noPadding' }}>
                <CollectionForm
                    hasWriteAccessForCollections={hasWriteAccessForCollections}
                    action={pageAction}
                    initialData={data}
                    useInlineDrawer={isLargeScreen}
                    showBreadcrumbs
                    appendTableLinkAction={() => {
                        /* TODO */
                    }}
                />
            </PageSection>
        </>
    );
}

export default CollectionsFormPage;
