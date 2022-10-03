import React, { CSSProperties, useCallback, useEffect, useState } from 'react';
import { Button, Divider, PageSection, Text, Title } from '@patternfly/react-core';

import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollection } from 'services/CollectionsService';
import CollectionForm from './CollectionForm';
import { CollectionPageAction } from './collections.utils';

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

function useLayoutSpaceObserver(
    root: HTMLElement | null | undefined,
    observationTargets: Element[],
    granularity = 10
) {
    const [usedSpace, setUsedSpace] = useState({ height: 0, width: 0 });

    useEffect(() => {
        function collectUsedSpace(intersectionEntries: IntersectionObserverEntry[]) {
            let height = 0;
            let width = 0;
            intersectionEntries.forEach(({ intersectionRect }) => {
                height += intersectionRect.height;
                width += intersectionRect.width;
            });

            if (height !== usedSpace.height || width !== usedSpace.width) {
                setUsedSpace({ height, width });
            }
        }

        const threshold = Array.from(Array(granularity + 1), (_, n) => n / granularity);
        const options = { root, rootMargin: '0px', threshold };
        const observer = new IntersectionObserver(collectUsedSpace, options);
        observationTargets.forEach((elem) => observer.observe(elem));

        return () => observer.disconnect();
    }, [root, observationTargets, granularity, usedSpace.height, usedSpace.width]);

    return usedSpace;
}

function CollectionsFormPage({ pageAction }: CollectionsFormPageProps) {
    const action = pageAction.type;
    const collectionId = action !== 'create' ? pageAction.collectionId : undefined;
    const collectionFetcher = useCallback(() => {
        return collectionId ? getCollection(collectionId) : noopRequest;
    }, [collectionId]);

    const { data, loading, error } = useRestQuery(collectionFetcher);

    const restrictedSpace = useLayoutSpaceObserver(
        document.getElementsByClassName('ob-target')[0]?.parentElement,
        Array.from(document.getElementsByClassName('ob-target'))
    );

    return (
        <>
            <PageSection variant="light" className="ob-target" padding={{ default: 'noPadding' }}>
                <PageSection className="pf-u-py-md">
                    <Text>Breadcrumbs</Text>
                </PageSection>
                <Divider component="div" />
                <PageSection>
                    <Title headingLevel="h1">
                        {data ? data.collection.name : 'Create collection'}
                    </Title>
                </PageSection>
            </PageSection>

            <Divider component="div" />
            <PageSection isFilled style={styleVarsForResultsList(restrictedSpace.height)}>
                <CollectionForm action={pageAction.type} collectionData={data} />
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" className="ob-target pf-u-flex-grow-0 pf-u-py-md">
                <Button className="pf-u-mr-md">{action} collection</Button>
                <Button variant="secondary">Cancel</Button>
            </PageSection>
        </>
    );
}

export default CollectionsFormPage;
