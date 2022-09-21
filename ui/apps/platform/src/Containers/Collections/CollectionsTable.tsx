import React, { useMemo } from 'react';
import { useHistory } from 'react-router-dom';
import {
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Pagination,
    Button,
    ButtonVariant,
    Truncate,
    SearchInput,
    Bullseye,
    Text,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import debounce from 'lodash/debounce';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
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
    hasWriteAccess: boolean;
};

// TODO We are using this value elsewhere in the app - extract to a central location?
const TYPING_DELAY = 800;

function CollectionsTable({
    collections,
    collectionsCount,
    pagination,
    searchFilter,
    setSearchFilter,
    getSortParams,
    hasWriteAccess,
}: CollectionsTableProps) {
    const history = useHistory();
    const { page, perPage, setPage, setPerPage } = pagination;
    const { selected, allRowsSelected, onSelect, onSelectAll } = useTableSelection(collections);
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
        () => debounce((value: string) => setSearchFilter({ Collection: value }), TYPING_DELAY),
        [setSearchFilter]
    );

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
                            placeholder="Search by name..."
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
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
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
                    {collections.map(({ id, name, description, inUse }, rowIndex) => {
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
                                onClick: () => {},
                                isDisabled: inUse,
                            },
                        ];

                        return (
                            <Tr key={id}>
                                {hasWriteAccess && (
                                    <Td
                                        select={{
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
        </>
    );
}

export default CollectionsTable;
