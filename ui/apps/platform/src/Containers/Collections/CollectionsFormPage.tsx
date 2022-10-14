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

const defaultCollectionData = {
    name: '',
    description: '',
    inUse: false,
    embeddedCollectionIds: [],
    selectorRules: {
        Deployment: null,
        Namespace: null,
        Cluster: null,
    },
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
    const initialData = data ? parseCollection(data.collection) : defaultCollectionData;

    let content: ReactElement | undefined;

    if (error) {
        content = (
            <>
                {error.message}
                {/* TODO - Handle UI for network errors */}
            </>
        );
    } else if (initialData instanceof AggregateError) {
        content = (
            <>
                {initialData.errors}
                {/* TODO - Handle UI for parse errors */}
            </>
        );
    } else if (loading && !initialData) {
        content = <>{/* TODO - Handle UI for loading state */}</>;
    } else if (initialData) {
        content = (
            <CollectionForm
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                initialData={initialData}
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
