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
    AlertActionCloseButton,
    AlertGroup,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import PageTitle from 'Components/PageTitle';
import LinkShim from 'Components/PatternFly/LinkShim';
import { collectionsPath } from 'routePaths';
import useRestQuery from 'Containers/Dashboard/hooks/useRestQuery';
import { deleteCollection, getCollectionCount, listCollections } from 'services/CollectionsService';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { Empty } from 'services/types';
import useToasts, { Toast } from 'hooks/patternfly/useToasts';
import CollectionsTable from './CollectionsTable';

type CollectionsTablePageProps = {
    hasWriteAccessForCollections: boolean;
};

const sortOptions = {
    sortFields: ['name', 'description', 'inUse'],
    defaultSortOption: { field: 'name', direction: 'asc' } as const,
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
    const isDataAvailable = typeof listData !== 'undefined' && typeof countData !== 'undefined';
    const isLoading = !isDataAvailable && (listLoading || countLoading);
    const loadError = listError || countError;

    /**
     * Deletes an array of collections by ids. Will alert individually for any deletion
     * requests that fail.
     */
    function onCollectionDelete(ids: string[]) {
        const promises: Promise<Empty>[] = [];
        ids.forEach((id) => {
            const deletionPromise = deleteCollection(id).request.catch((err) => {
                addToast(`Could not delete collection ${id}`, 'danger', err.message);
                return Promise.reject(err);
            });
            promises.push(deletionPromise);
        });

        return Promise.allSettled(promises).then((promiseResults) => {
            const totalDeleted = promiseResults.filter((res) => res.status === 'fulfilled').length;
            const collectionText = pluralize('collection', ids.length);

            if (totalDeleted > 0 && totalDeleted === ids.length) {
                // All collections deleted successfully
                addToast(
                    `Successfully deleted ${totalDeleted} selected ${collectionText}`,
                    'success'
                );
                // Some, but not all, deletion requests failed
            } else if (totalDeleted > 0) {
                addToast(
                    `Deleted ${totalDeleted} of ${ids.length} selected ${collectionText}`,
                    'warning'
                );
            }

            listRefetch();
            countRefetch();
        });
    }

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
                    collectionsCount={countData}
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
