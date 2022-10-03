import React, { useCallback } from 'react';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import { Button, Divider, PageSection, Title } from '@patternfly/react-core';
import CollectionForm from './CollectionForm';
import { CollectionPageAction } from './collections.utils';

export type CollectionsFormPageProps = {
    pageAction: CollectionPageAction;
};

const noopRequest = {
    request: Promise.resolve(undefined),
    cancel: () => {},
};

function CollectionsFormPage({ pageAction }: CollectionsFormPageProps) {
    const action = pageAction.type;
    const collectionId = action !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(() => {
        return collectionId ? getCollection(collectionId) : noopRequest;
    }, [collectionId]);

    const { data, loading, error } = useRestQuery(collectionFetcher);

    return (
        <>
            <PageSection variant="light" className="pf-u-py-md">
                Breadcrumbs
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Title headingLevel="h1">{data ? data.collection.name : 'Create collection'}</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled>
                <CollectionForm action={pageAction.type} collectionData={data} />
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" className="pf-u-flex-grow-0 pf-u-py-md">
                <Button className="pf-u-mr-md">{action} collection</Button>
                <Button variant="secondary">Cancel</Button>
            </PageSection>
        </>
    );
}

export default CollectionsFormPage;
