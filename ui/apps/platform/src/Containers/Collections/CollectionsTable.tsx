import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
    Button,
    Pagination,
    SearchInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Truncate,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { GetSortParams } from 'hooks/useURLSort';
import { Collection } from 'services/CollectionsService';
import { SearchFilter } from 'types/search';
import { collectionsBasePath } from 'routePaths';
import { getTableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';

export type CollectionsTableProps = {
    isLoading: boolean;
    error: Error | undefined;
    collections: Collection[];
    collectionsCount: number;
    pagination: UseURLPaginationResult;
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
    getSortParams: GetSortParams;
    onCollectionDelete: (collection: Collection) => Promise<void>;
    hasWriteAccess: boolean;
};

function CollectionsTable({
    isLoading,
    error,
    collections,
    collectionsCount,
    pagination,
    searchFilter,
    setSearchFilter,
    getSortParams,
    onCollectionDelete,
    hasWriteAccess,
}: CollectionsTableProps) {
    const navigate = useNavigate();
    const { page, perPage, setPage, setPerPage } = pagination;
    const [isDeleting, setIsDeleting] = useState(false);
    const [collectionToDelete, setCollectionToDelete] = useState<Collection | null>(null);
    const [searchValue, setSearchValue] = useState(() => {
        const filter = searchFilter['Collection Name'];
        return Array.isArray(filter) ? filter.join(',') : filter;
    });
    const hasCollections = collections.length > 0;

    function onSearchInputChange(_event, value) {
        setSearchValue(value);
    }

    function getEnabledSortParams(field: string) {
        return hasCollections ? getSortParams(field) : undefined;
    }

    function onEditCollection(id: string) {
        navigate(`${collectionsBasePath}/${id}?action=edit`);
    }

    function onCloneCollection(id: string) {
        navigate(`${collectionsBasePath}/${id}?action=clone`);
    }

    function onConfirmDeleteCollection(collection: Collection) {
        setIsDeleting(true);
        onCollectionDelete(collection)
            .catch(() => {
                // TODO render error in dialog and move finally code to then block.
            })
            .finally(() => {
                setCollectionToDelete(null);
                setIsDeleting(false);
            });
    }

    function onCancelDeleteCollection() {
        setCollectionToDelete(null);
    }

    const tableState = getTableUIState({
        isLoading,
        data: collections,
        error,
        searchFilter,
    });

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="search-filter" className="pf-v5-u-flex-grow-1">
                        <SearchInput
                            aria-label="Search by name"
                            placeholder="Search by name"
                            value={searchValue}
                            onChange={onSearchInputChange}
                            onSearch={() => setSearchFilter({ 'Collection Name': searchValue })}
                            onClear={() => {
                                setSearchValue('');
                                setSearchFilter({});
                            }}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            isCompact
                            itemCount={collectionsCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Table>
                <Thead>
                    <Tr>
                        <Th modifier="wrap" sort={getEnabledSortParams('Collection Name')}>
                            Collection
                        </Th>
                        <Th modifier="wrap">Description</Th>
                        {hasWriteAccess && (
                            <Th>
                                <span className="pf-v5-screen-reader">Row actions</span>
                            </Th>
                        )}
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={hasWriteAccess ? 3 : 2}
                    errorProps={{
                        title: 'There was an error loading the collections',
                    }}
                    emptyProps={{
                        message: 'You have not created any collections yet',
                        children: (
                            <div>
                                <Button
                                    variant="primary"
                                    component={LinkShim}
                                    href={`${collectionsBasePath}?action=create`}
                                >
                                    Create collection
                                </Button>
                            </div>
                        ),
                    }}
                    filteredEmptyProps={{
                        title: 'No collections found',
                        message: 'Clear all filters and try again',
                        onClearFilters: () => {
                            setSearchFilter({});
                            setSearchValue('');
                        },
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((collection) => {
                                const { id, name, description } = collection;
                                const actionItems = [
                                    {
                                        title: 'Edit collection',
                                        onClick: () => onEditCollection(id),
                                    },
                                    {
                                        title: 'Clone collection',
                                        onClick: () => onCloneCollection(id),
                                    },
                                    {
                                        isSeparator: true,
                                    },
                                    {
                                        title: 'Delete collection',
                                        onClick: () => setCollectionToDelete(collection),
                                    },
                                ];

                                return (
                                    <Tr key={id}>
                                        <Td dataLabel="Collection">
                                            <Link to={`${collectionsBasePath}/${id}`}>{name}</Link>
                                        </Td>
                                        <Td dataLabel="Description">
                                            <Truncate
                                                content={description || '-'}
                                                tooltipPosition="top"
                                            />
                                        </Td>
                                        {hasWriteAccess && <Td actions={{ items: actionItems }} />}
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
            {collectionToDelete && (
                <ConfirmationModal
                    ariaLabel="Confirm delete"
                    confirmText="Delete collection"
                    isLoading={isDeleting}
                    isOpen
                    onConfirm={() => onConfirmDeleteCollection(collectionToDelete)}
                    onCancel={onCancelDeleteCollection}
                >
                    Are you sure you want to delete &lsquo;{collectionToDelete.name}&rsquo;?
                </ConfirmationModal>
            )}
        </>
    );
}

export default CollectionsTable;
