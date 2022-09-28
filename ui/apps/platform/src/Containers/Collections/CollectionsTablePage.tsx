import React, { useCallback } from 'react';
import {
    PageSection,
    Title,
    Text,
    Button,
    Flex,
    FlexItem,
    ButtonVariant,
    Divider,
    Alert,
    Bullseye,
    Spinner,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsPath } from 'routePaths';
import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { getCollectionCount, listCollections } from 'services/CollectionsService';
import useURLSort from 'hooks/useURLSort';
import CollectionsTable from './CollectionsTable';

type CollectionsTablePageProps = {
    hasWriteAccessForCollections: boolean;
};

const sortOptions = {
    sortFields: ['name', 'description', 'inUse'],
    defaultSortOption: { field: 'name', direction: 'asc' } as const,
};

function CollectionsTablePage({ hasWriteAccessForCollections }: CollectionsTablePageProps) {
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const listQuery = useCallback(() => listCollections({}, sortOption, 0, 20), [sortOption]);
    const { data: listData, loading: listLoading, error: listError } = useRestQuery(listQuery);

    const countQuery = useCallback(() => getCollectionCount({}), []);
    const { data: countData, loading: countLoading, error: countError } = useRestQuery(countQuery);

    const isDataAvailable = typeof listData !== 'undefined' && typeof countData !== 'undefined';
    const isLoading = !isDataAvailable && (listLoading || countLoading);
    const loadError = listError || countError;

    let pageContent = (
        <PageSection variant="light" isFilled>
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        </PageSection>
    );

    if (loadError) {
        pageContent = (
            <PageSection variant="light" isFilled>
                <Bullseye>
                    <Alert variant="danger" title={loadError.message} />
                </Bullseye>
            </PageSection>
        );
    }

    if (isDataAvailable && !isLoading && !loadError) {
        pageContent = (
            <PageSection>
                <CollectionsTable
                    collections={listData}
                    getSortParams={getSortParams}
                    hasWriteAccess={hasWriteAccessForCollections}
                />
            </PageSection>
        );
    }

    return (
        <>
            <PageTitle title="Collections" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Collections</Title>
                        <Text>
                            Configure deployment collections to associate with other workflows
                        </Text>
                    </FlexItem>
                    {hasWriteAccessForCollections && (
                        <FlexItem align={{ default: 'alignRight' }}>
                            <Button
                                variant={ButtonVariant.primary}
                                component={LinkShim}
                                href={`${collectionsPath}?action=create`}
                            >
                                Create collection
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            {pageContent}
        </>
    );
}

export default CollectionsTablePage;
