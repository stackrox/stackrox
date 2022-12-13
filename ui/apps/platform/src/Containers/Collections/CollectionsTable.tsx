import React, { useMemo, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Bullseye,
    Button,
    ButtonVariant,
    Pagination,
    SearchInput,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Truncate,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import debounce from 'lodash/debounce';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { GetSortParams } from 'hooks/useURLSort';
import { Collection } from 'services/CollectionsService';
import { SearchFilter } from 'types/search';
import { collectionsBasePath } from 'routePaths';

export type CollectionsTableProps = {
    collections: Collection[];
    collectionsCount: number;
    pagination: UseURLPaginationResult;
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
    getSortParams: GetSortParams;
    onCollectionDelete: (collection: Collection) => Promise<void>;
    hasWriteAccess: boolean;
};

const SEARCH_INPUT_REQUEST_DELAY = 800;

function CollectionsTable({
    collections,
    collectionsCount,
    pagination,
    searchFilter,
    setSearchFilter,
    getSortParams,
    onCollectionDelete,
    hasWriteAccess,
}: CollectionsTableProps) {
    const history = useHistory();
    const { page, perPage, setPage, setPerPage } = pagination;
    const [isDeleting, setIsDeleting] = useState(false);
    const [collectionToDelete, setCollectionToDelete] = useState<Collection | null>(null);
    const hasCollections = collections.length > 0;

    function getEnabledSortParams(field: string) {
        return hasCollections ? getSortParams(field) : undefined;
    }

    function onEditCollection(id: string) {
        history.push({
            pathname: `${collectionsBasePath}/${id}`,
            search: 'action=edit',
        });
    }

    function onCloneCollection(id: string) {
        history.push({
            pathname: `${collectionsBasePath}/${id}`,
            search: 'action=clone',
        });
    }

    const onSearchInputChange = useMemo(
        () =>
            debounce(
                (value: string) => setSearchFilter({ 'Collection Name': value }),
                SEARCH_INPUT_REQUEST_DELAY
            ),
        [setSearchFilter]
    );

    function onConfirmDeleteCollection(collection: Collection) {
        setIsDeleting(true);
        onCollectionDelete(collection).finally(() => {
            setCollectionToDelete(null);
            setIsDeleting(false);
        });
    }

    function onCancelDeleteCollection() {
        setCollectionToDelete(null);
    }

    // Currently, it is not expected that the value of `searchFilter.Collection` will
    // be an array even though it would valid. This is a safeguard for future code
    // changes that might change this assumption.
    const searchValue = Array.isArray(searchFilter.Collection)
        ? searchFilter.Collection.join('+')
        : searchFilter.Collection;

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="search-filter" className="pf-u-flex-grow-1">
                        <SearchInput
                            aria-label="Search by name"
                            placeholder="Search by name"
                            value={searchValue}
                            onChange={onSearchInputChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            isCompact
                            itemCount={collectionsCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                if (collectionsCount < (page - 1) * newPerPage) {
                                    setPage(1);
                                }
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <TableComposable variant={TableVariant.compact}>
                <Thead>
                    <Tr>
                        <Th
                            modifier="wrap"
                            width={25}
                            sort={getEnabledSortParams('Collection Name')}
                        >
                            Collection
                        </Th>
                        <Th modifier="wrap">Description</Th>
                        <Th modifier="wrap" width={10}>
                            In use
                        </Th>
                        <Th aria-label="Row actions" />
                    </Tr>
                </Thead>
                <Tbody>
                    {hasCollections || (
                        <Tr>
                            <Td colSpan={hasWriteAccess ? 5 : 3}>
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No collections found"
                                        headingLevel="h2"
                                        icon={SearchIcon}
                                    >
                                        <Text>Clear all filters and try again.</Text>
                                        <Button variant="link" onClick={() => setSearchFilter({})}>
                                            Clear all filters
                                        </Button>
                                    </EmptyStateTemplate>
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {collections.map((collection) => {
                        const { id, name, description, inUse } = collection;
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
                                title: inUse ? 'Cannot delete (in use)' : 'Delete collection',
                                onClick: () => setCollectionToDelete(collection),
                                isDisabled: inUse,
                            },
                        ];

                        return (
                            <Tr key={id}>
                                <Td dataLabel="Collection">
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={`${collectionsBasePath}/${id}`}
                                    >
                                        {name}
                                    </Button>
                                </Td>
                                <Td dataLabel="Description">
                                    <Truncate content={description || '-'} tooltipPosition="top" />
                                </Td>
                                <Td dataLabel="In Use">{inUse ? 'Yes' : 'No'}</Td>
                                {hasWriteAccess && <Td actions={{ items: actionItems }} />}
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            {collectionToDelete && (
                <ConfirmationModal
                    ariaLabel="Confirm delete"
                    confirmText="Delete"
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
