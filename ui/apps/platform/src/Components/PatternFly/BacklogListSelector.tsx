import React from 'react';
import type { Key, ReactElement, ReactNode } from 'react';
import {
    Badge,
    Button,
    EmptyState,
    EmptyStateIcon,
    Flex,
    FormGroup,
    EmptyStateHeader,
    EmptyStateFooter,
    Icon,
} from '@patternfly/react-core';
import { CubesIcon, MinusCircleIcon, PlusCircleIcon } from '@patternfly/react-icons';

import { Table, Tbody, Td, Tr } from '@patternfly/react-table';
import type { BaseCellProps } from '@patternfly/react-table';

type BacklogTableProps<Item> = {
    type: 'selected' | 'deselected';
    label: string | undefined;
    items: Item[];
    listAction: (item: Item) => void;
    rowKey: (item: Item) => Key;
    cells: {
        name: string;
        render: (item: Item) => ReactNode;
        width?: BaseCellProps['width'];
    }[];
    buttonText: string;
    searchFilter?: (item: Item) => boolean;
    showBadge: boolean;
};

function BacklogTable<Item>({
    type,
    label,
    items,
    listAction,
    rowKey,
    cells,
    buttonText,
    searchFilter = () => true,
    showBadge,
}: BacklogTableProps<Item>): ReactElement {
    const actionIcon =
        type === 'selected' ? (
            <Icon>
                <MinusCircleIcon color="var(--pf-v5-global--danger-color--200)" />
            </Icon>
        ) : (
            <Icon>
                <PlusCircleIcon color="var(--pf-v5-global--primary-color--100)" />
            </Icon>
        );

    const itemsToDisplay = items.filter(searchFilter);

    return (
        <FormGroup
            label={
                <>
                    {label}
                    {showBadge && (
                        <Badge className="pf-v5-u-ml-sm" isRead>
                            {items.length}
                        </Badge>
                    )}
                </>
            }
        >
            {itemsToDisplay.length > 0 ? (
                <Table aria-label={label}>
                    <Tbody>
                        {itemsToDisplay.map((item) => (
                            <Tr key={rowKey(item)}>
                                {cells.map(({ name, width, render }) => (
                                    <Td key={name} dataLabel={name} width={width}>
                                        {render(item)}
                                    </Td>
                                ))}
                                <Td width={10} dataLabel="Item action">
                                    <Button
                                        variant="link"
                                        onClick={() => listAction(item)}
                                        icon={actionIcon}
                                        className="pf-v5-u-text-nowrap"
                                        isInline
                                    >
                                        {buttonText}
                                    </Button>
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                </Table>
            ) : (
                <EmptyState variant="xs">
                    <EmptyStateHeader icon={<EmptyStateIcon icon={CubesIcon} />} />
                    <EmptyStateFooter>
                        <p>No items remaining</p>
                    </EmptyStateFooter>
                </EmptyState>
            )}
        </FormGroup>
    );
}

export type BacklogListSelectorProps<Item> = {
    selectedOptions: Item[];
    deselectedOptions: Item[];
    onSelectItem: (item: Item) => void;
    onDeselectItem: (item: Item) => void;
    onSelectionChange?: (selected: Item[], deselected: Item[]) => void;
    rowKey: (item: Item) => Key;
    cells: {
        name: string;
        render: (item: Item) => ReactNode;
        width?: BaseCellProps['width'];
    }[];
    selectedLabel?: string;
    deselectedLabel?: string;
    selectButtonText?: string;
    deselectButtonText?: string;
    searchFilter?: (item: Item) => boolean;
    showBadge?: boolean;
};

function BacklogListSelector<Item>({
    selectedOptions,
    deselectedOptions,
    onSelectItem,
    onDeselectItem,
    onSelectionChange = () => {},
    rowKey,
    cells,
    selectedLabel = 'Selected items',
    deselectedLabel = 'Deselected items',
    selectButtonText = 'Add',
    deselectButtonText = 'Remove',
    searchFilter,
    showBadge = false,
}: BacklogListSelectorProps<Item>) {
    function onSelect(item: Item) {
        onSelectItem(item);
        onSelectionChange(
            selectedOptions.concat(item),
            deselectedOptions.filter((option) => option !== item)
        );
    }

    function onDeselect(item: Item) {
        onDeselectItem(item);
        onSelectionChange(
            selectedOptions.filter((option) => option !== item),
            deselectedOptions.concat(item)
        );
    }

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXl' }}>
            <BacklogTable
                type="selected"
                label={selectedLabel}
                items={selectedOptions}
                listAction={onDeselect}
                buttonText={deselectButtonText}
                rowKey={rowKey}
                cells={cells}
                searchFilter={searchFilter}
                showBadge={showBadge}
            />
            <BacklogTable
                type="deselected"
                label={deselectedLabel}
                items={deselectedOptions}
                listAction={onSelect}
                rowKey={rowKey}
                buttonText={selectButtonText}
                cells={cells}
                searchFilter={searchFilter}
                showBadge={showBadge}
            />
        </Flex>
    );
}

export default BacklogListSelector;
