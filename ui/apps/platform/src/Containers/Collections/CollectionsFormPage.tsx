import React, { ReactElement, useCallback } from 'react';
import { PageSection } from '@patternfly/react-core';
import { useMediaQuery } from 'react-responsive';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import {
    CollectionResponse,
    getCollection,
    listCollections,
    ResolvedCollectionResponse,
} from 'services/CollectionsService';
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
    embeddedCollections: [],
    resourceSelectors: {
        Deployment: {},
        Namespace: {},
        Cluster: {},
    },
};

function getEmbeddedCollections({ collection }: ResolvedCollectionResponse): Promise<{
    collection: CollectionResponse;
    embeddedCollections: CollectionResponse[];
}> {
    if (collection.embeddedCollections.length === 0) {
        return Promise.resolve({ collection, embeddedCollections: [] });
    }
    const idSearchString = collection.embeddedCollections.map(({ id }) => id).join(',');
    const searchFilter = { 'Collection ID': idSearchString };
    const { request } = listCollections(searchFilter, { field: 'name', reversed: false });
    return request.then((embeddedCollections) => ({ collection, embeddedCollections }));
}

function CollectionsFormPage({
    hasWriteAccessForCollections,
    pageAction,
}: CollectionsFormPageProps) {
    const isLargeScreen = useMediaQuery({ query: '(min-width: 992px)' }); // --pf-global--breakpoint--lg
    const collectionId = pageAction.type !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(() => {
        if (!collectionId) {
            return noopRequest;
        }
        const { request, cancel } = getCollection(collectionId);
        return { request: request.then(getEmbeddedCollections), cancel };
    }, [collectionId]);
    const { data, loading, error } = useRestQuery(collectionFetcher);
    const initialData = data ? parseCollection(data.collection) : defaultCollectionData;
    const initialEmbeddedCollections = data ? data.embeddedCollections : [];

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
    } else if (loading) {
        content = <>{/* TODO - Handle UI for loading state */}</>;
    } else if (initialData) {
        content = (
            <CollectionForm
                hasWriteAccessForCollections={hasWriteAccessForCollections}
                action={pageAction}
                initialData={initialData}
                initialEmbeddedCollections={initialEmbeddedCollections}
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
