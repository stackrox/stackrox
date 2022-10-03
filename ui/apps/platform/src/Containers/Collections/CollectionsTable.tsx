import React, { useMemo, useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Bullseye,
    Button,
    ButtonVariant,
    Dropdown,
    DropdownItem,
    DropdownToggle,
    Pagination,
    SearchInput,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    ToolbarItemVariant,
    Truncate,
} from '@patternfly/react-core';
import { CaretDownIcon, SearchIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import debounce from 'lodash/debounce';
import pluralize from 'pluralize';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useTableSelection from 'hooks/useTableSelection';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { GetSortParams } from 'hooks/useURLSort';
import { CollectionResponse } from 'services/CollectionsService';
import { SearchFilter } from 'types/search';
import { collectionsPath } from 'routePaths';

export type CollectionsTableProps = {
    collections: CollectionResponse[];
    collectionsCount: number;
    pagination: UseURLPaginationResult;
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
    getSortParams: GetSortParams;
    onCollectionDelete: (ids: string[]) => Promise<void>;
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
    const { isOpen, onToggle, closeSelect } = useSelectToggle();
    const { selected, allRowsSelected, hasSelections, onSelect, onSelectAll, getSelectedIds } =
        useTableSelection(collections);
    const [isDeleting, setIsDeleting] = useState(false);
    const [deletingIds, setDeletingIds] = useState<string[]>([]);
    const hasCollections = collections.length > 0;

    function getEnabledSortParams(field: string) {
        return hasCollections ? getSortParams(field) : undefined;
    }

    function onEditCollection(id: string) {
        history.push({
            pathname: `${collectionsPath}/${id}`,
            search: 'action=edit',
        });
    }

    function onCloneCollection(id: string) {
        history.push({
            pathname: `${collectionsPath}/${id}`,
            search: 'action=clone',
        });
    }

    const onSearchInputChange = useMemo(
        () =>
            debounce(
                (value: string) => setSearchFilter({ Collection: value }),
                SEARCH_INPUT_REQUEST_DELAY
            ),
        [setSearchFilter]
    );

    function onConfirmDeleteCollection() {
        setIsDeleting(true);
        onCollectionDelete(deletingIds).finally(() => {
            setDeletingIds([]);
            setIsDeleting(false);
        });
    }

    function onCancelDeleteCollection() {
        setDeletingIds([]);
    }

    const unusedSelectedCollectionIds = collections
        .filter((c) => getSelectedIds().includes(c.id) && !c.inUse)
        .map((c) => c.id);

    // A map to keep track of row index within the table to the collection id
    // for checkbox selection after the table has been sorted.
    const rowIdToIndex = {};
    collections.forEach(({ id }, idx) => {
        rowIdToIndex[id] = idx;
    });

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
                    {hasWriteAccess && (
                        <>
                            <ToolbarItem variant={ToolbarItemVariant.separator} />
                            <ToolbarItem className="pf-u-flex-grow-1">
                                <Dropdown
                                    onSelect={closeSelect}
                                    toggle={
                                        <DropdownToggle
                                            isDisabled={!hasSelections}
                                            isPrimary
                                            onToggle={onToggle}
                                            toggleIndicator={CaretDownIcon}
                                        >
                                            Bulk actions
                                        </DropdownToggle>
                                    }
                                    isOpen={isOpen}
                                    dropdownItems={[
                                        <DropdownItem
                                            key="Delete collection"
                                            component="button"
                                            isDisabled={unusedSelectedCollectionIds.length === 0}
                                            onClick={() => {
                                                setDeletingIds(unusedSelectedCollectionIds);
                                            }}
                                        >
                                            {unusedSelectedCollectionIds.length > 0
                                                ? `Delete collections (${unusedSelectedCollectionIds.length})`
                                                : 'Cannot delete (in use)'}
                                        </DropdownItem>,
                                    ]}
                                />
                            </ToolbarItem>
                        </>
                    )}
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
                        {hasWriteAccess && (
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                        )}
                        <Th modifier="wrap" width={25} sort={getEnabledSortParams('name')}>
                            Collection
                        </Th>
                        <Th modifier="wrap" sort={getEnabledSortParams('description')}>
                            Description
                        </Th>
                        <Th modifier="wrap" width={10} sort={getEnabledSortParams('inUse')}>
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
                    {collections.map(({ id, name, description, inUse }) => {
                        const rowIndex = rowIdToIndex[id];
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
                                onClick: () => setDeletingIds([id]),
                                isDisabled: inUse,
                            },
                        ];

                        return (
                            <Tr key={id}>
                                {hasWriteAccess && (
                                    <Td
                                        title={inUse ? 'Collection is in use' : ''}
                                        select={{
                                            disable: inUse,
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                )}
                                <Td dataLabel="Collection">
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={`${collectionsPath}/${id}`}
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
            <ConfirmationModal
                ariaLabel="Confirm delete"
                confirmText="Delete"
                isLoading={isDeleting}
                isOpen={deletingIds.length !== 0}
                onConfirm={onConfirmDeleteCollection}
                onCancel={onCancelDeleteCollection}
            >
                Are you sure you want to delete {deletingIds.length}&nbsp;
                {pluralize('collection', deletingIds.length)}?
            </ConfirmationModal>
        </>
    );
}

export default CollectionsTable;
