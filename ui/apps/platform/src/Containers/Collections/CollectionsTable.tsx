import React from 'react';
import { useHistory } from 'react-router-dom';
import { Button, ButtonVariant, Truncate, Bullseye, Text } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { TableComposable, TableVariant, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import LinkShim from 'Components/PatternFly/LinkShim';
import useTableSelection from 'hooks/useTableSelection';
import { CollectionResponse } from 'services/CollectionsService';
import { GetSortParams } from 'hooks/useURLSort';
import { collectionsPath } from 'routePaths';

export type CollectionsTableProps = {
    collections: CollectionResponse[];
    getSortParams: GetSortParams;
    hasWriteAccess: boolean;
};

function CollectionsTable({ collections, getSortParams, hasWriteAccess }: CollectionsTableProps) {
    const history = useHistory();
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

    return (
        <>
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
                                        <Button variant="link" onClick={() => {}}>
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
