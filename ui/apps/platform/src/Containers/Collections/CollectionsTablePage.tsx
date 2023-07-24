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
    AlertActionCloseButton,
    AlertGroup,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsBasePath } from 'routePaths';
import useRestQuery from 'hooks/useRestQuery';
import {
    Collection,
    deleteCollection,
    getCollectionCount,
    listCollections,
} from 'services/CollectionsService';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import CollectionsTable from './CollectionsTable';

type CollectionsTablePageProps = {
    hasWriteAccessForCollections: boolean;
};

const sortOptions = {
    sortFields: ['Collection Name'],
    defaultSortOption: { field: 'Collection Name', direction: 'asc' } as const,
};

function CollectionsTablePage({ hasWriteAccessForCollections }: CollectionsTablePageProps) {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(20);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { toasts, addToast, removeToast } = useToasts();

    const listQuery = useCallback(
        () => listCollections(searchFilter, sortOption, page - 1, perPage),
        [searchFilter, sortOption, page, perPage]
    );
    const {
        data: listData,
        loading: listLoading,
        error: listError,
        refetch: listRefetch,
    } = useRestQuery(listQuery);

    const countQuery = useCallback(() => getCollectionCount(searchFilter), [searchFilter]);
    const {
        data: countData,
        loading: countLoading,
        error: countError,
        refetch: countRefetch,
    } = useRestQuery(countQuery);

    const isLoading = listLoading || countLoading;
    const loadError = listError || countError;

    function onCollectionDelete({ id, name }: Collection) {
        const { request } = deleteCollection(id);

        return request
            .then(() => {
                addToast(`Successfully deleted '${name}'`, 'success');
                listRefetch();
                countRefetch();
            })
            .catch((err) => {
                const error = getAxiosErrorMessage(err);
                addToast(`Could not delete collection '${name}'`, 'danger', error);
            });
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
                                href={`${collectionsBasePath}?action=create`}
                            >
                                Create collection
                            </Button>
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <CollectionsTable
                    isLoading={isLoading}
                    error={loadError}
                    collections={listData ?? []}
                    collectionsCount={countData ?? 0}
                    pagination={pagination}
                    searchFilter={searchFilter}
                    setSearchFilter={(value) => {
                        setPage(1);
                        setSearchFilter(value);
                    }}
                    getSortParams={getSortParams}
                    onCollectionDelete={onCollectionDelete}
                    hasWriteAccess={hasWriteAccessForCollections}
                />
            </PageSection>
            <AlertGroup isToast isLiveRegion>
                {toasts.map(({ key, variant, title, children }: Toast) => (
                    <Alert
                        key={key}
                        variant={variant}
                        title={title}
                        timeout
                        onTimeout={() => removeToast(key)}
                        actionClose={
                            <AlertActionCloseButton
                                title={title}
                                variantLabel={variant}
                                onClose={() => removeToast(key)}
                            />
                        }
                    >
                        {children}
                    </Alert>
                ))}
            </AlertGroup>
        </>
    );
}

export default CollectionsTablePage;
