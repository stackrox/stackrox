import React, { ReactElement, useCallback } from 'react';
import { PageSection } from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import { CollectionPageAction } from './collections.utils';
import CollectionForm from './CollectionForm';
import { Collection } from './types';

export type CollectionsFormPageProps = {
    hasWriteAccessForCollections: boolean;
    pageAction: CollectionPageAction;
};

const emptyCollection: Collection = {
    name: '',
    description: '',
    inUse: false,
    embeddedCollectionIds: [],
    selectorRules: {
        Deployment: {},
        Namespace: {},
        Cluster: {},
    },
};
const noopRequest = {
    request: Promise.resolve(emptyCollection),
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
    const { data, loading, error } = useRestQuery<Collection, AggregateError>(collectionFetcher);

    let content: ReactElement | undefined;

    if (error) {
        content = <>{/* TODO - Handle UI for parse errors */}</>;
    }

    if (loading) {
        content = <>{/* TODO - Handle UI for loading state */}</>;
    }

    if (data) {
        content = (
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
