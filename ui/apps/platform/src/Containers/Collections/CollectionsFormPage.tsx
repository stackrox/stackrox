import React, { ReactElement, useCallback } from 'react';
import { PageSection } from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import CollectionForm from './CollectionForm';
import { parseCollection } from './parser';

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
    const collectionId = pageAction.type !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(
        () => (collectionId ? getCollection(collectionId) : noopRequest),
        [collectionId]
    );
    const { data, loading, error } = useRestQuery(collectionFetcher);
    const collection = data ? parseCollection(data.collection) : undefined;

    let content: ReactElement | undefined;

    if (error) {
        content = <>{/* TODO - Handle UI for network errors */}</>;
    } else if (collection instanceof AggregateError) {
        content = <>{/* TODO - Handle UI for parse errors */}</>;
    } else if (loading && !collection) {
        content = <>{/* TODO - Handle UI for loading state */}</>;
    } else if (collection) {
        content = (
            <CollectionForm
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                initialData={collection}
                useInlineDrawer={isLargeScreen}
                showBreadcrumbs
                appendTableLinkAction={() => {
                    /* TODO */
                }}
            />
        );
    }

    return (
        <>
            <PageSection className="pf-u-h-100" padding={{ default: 'noPadding' }}>
                {content}
            </PageSection>
        </>
    );
}

export default CollectionsFormPage;
