import React, { CSSProperties, useCallback } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Divider,
    PageSection,
    Title,
} from '@patternfly/react-core';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import useLayoutSpaceObserver from 'hooks/useLayoutSpaceObserver';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import { collectionsBasePath } from 'routePaths';
import { CollectionPageAction } from './collections.utils';
import CollectionForm from './CollectionForm';

export type CollectionsFormPageProps = {
    pageAction: CollectionPageAction;
};

const noopRequest = {
    request: Promise.resolve(undefined),
    cancel: () => {},
};

function styleVarsForResultsList(resultListSpaceOffset: number): CSSProperties {
    return {
        '--collection-results-container-max-height': `calc(
            100vh -
            var(--pf-c-page__header--MinHeight) -
            var(--pf-c-page__main-section--PaddingTop) -
            var(--pf-c-page__main-section--PaddingBottom) -
            ${resultListSpaceOffset}px
        )`,
    } as CSSProperties;
}

const observedClass = 'collections-observe-layout';

function CollectionsFormPage({ pageAction }: CollectionsFormPageProps) {
    const action = pageAction.type;
    const collectionId = action !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(() => {
        return collectionId ? getCollection(collectionId) : noopRequest;
    }, [collectionId]);

    const { data } = useRestQuery(collectionFetcher);

    const watchElements = Array.from(document.getElementsByClassName(observedClass));
    const { height } = useLayoutSpaceObserver(watchElements[0]?.parentElement, watchElements);

    return (
        <>
            <PageSection
                variant="light"
                className={observedClass}
                padding={{ default: 'noPadding' }}
            >
                <PageSection className="pf-u-py-md">
                    <Breadcrumb>
                        <BreadcrumbItemLink to={collectionsBasePath}>
                            Collections
                        </BreadcrumbItemLink>
                        <BreadcrumbItem>
                            {data ? data.collection.name : 'Create collection'}
                        </BreadcrumbItem>
                    </Breadcrumb>
                </PageSection>
                <Divider component="div" />
                <PageSection>
                    <Title headingLevel="h1">
                        {data ? data.collection.name : 'Create collection'}
                    </Title>
                </PageSection>
            </PageSection>

            <Divider component="div" />
            <PageSection isFilled style={styleVarsForResultsList(height)}>
                <CollectionForm />
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" className={`${observedClass} pf-u-flex-grow-0 pf-u-py-md`}>
                <Button className="pf-u-mr-md">{action} collection</Button>
                <Button variant="secondary">Cancel</Button>
            </PageSection>
        </>
    );
}

export default CollectionsFormPage;
