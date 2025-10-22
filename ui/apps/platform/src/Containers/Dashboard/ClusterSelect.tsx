import React, { useState } from 'react';
import type { MouseEvent, Ref } from 'react';
import {
    Badge,
    Divider,
    Flex,
    FlexItem,
    MenuToggle,
    SearchInput,
    Select,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import { flattenFilterValue } from 'utils/searchUtils';
import type { Cluster } from './types';

// TODO: Refactor ClusterSelect and NamespaceSelect to use a shared reusable component
const SELECT_ALL = '##SELECT_ALL##';

export type SelectionChangeAction =
    | { type: 'add'; value: string; selection: string[] }
    | { type: 'remove'; value: string; selection: string[] };

type ClusterSelectProps = {
    clusters: Cluster[];
    clusterSearch: string | string[] | undefined;
    isDisabled?: boolean;
    onChange: (selectionChangeAction: SelectionChangeAction) => void;
    onSelectAll: () => void;
};

function ClusterSelect({
    clusters,
    clusterSearch,
    isDisabled = false,
    onChange,
    onSelectAll,
}: ClusterSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const currentSelection = flattenFilterValue(clusterSearch, SELECT_ALL);

    const onToggle = () => {
        const willBeOpen = !isOpen;
        setIsOpen(willBeOpen);
        if (!willBeOpen) {
            setFilterValue('');
        }
    };

    function onSelect(_event: MouseEvent | undefined, selectedTarget: string | number | undefined) {
        if (selectedTarget === SELECT_ALL) {
            onSelectAll();
        } else if (typeof selectedTarget !== 'string') {
            // Do nothing for invalid types
        } else if (currentSelection === SELECT_ALL) {
            onChange({ type: 'add', value: selectedTarget, selection: [selectedTarget] });
        } else {
            const isRemoval = currentSelection.includes(selectedTarget);
            const selection = isRemoval
                ? currentSelection.filter((cs) => cs !== selectedTarget)
                : currentSelection.concat(selectedTarget);

            // If deselecting the last cluster, revert to "All clusters" selected
            if (selection.length === 0) {
                onSelectAll();
            } else {
                onChange({
                    type: isRemoval ? 'remove' : 'add',
                    value: selectedTarget,
                    selection,
                });
            }
        }
    }

    const filteredClusters = filterValue
        ? clusters.filter(({ name }) => name.toLowerCase().includes(filterValue.toLowerCase()))
        : clusters;

    const toggle = (toggleRef: Ref<MenuToggleElement>) => {
        const numSelected = currentSelection === SELECT_ALL ? 0 : currentSelection.length;
        const placeholderText = currentSelection === SELECT_ALL ? 'All clusters' : 'Clusters';

        return (
            <MenuToggle
                ref={toggleRef}
                onClick={onToggle}
                isExpanded={isOpen}
                isDisabled={isDisabled}
                style={{ width: '180px' }}
                aria-label="Select clusters"
            >
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <FlexItem>{placeholderText}</FlexItem>
                    {numSelected > 0 && <Badge isRead>{numSelected}</Badge>}
                </Flex>
            </MenuToggle>
        );
    };

    return (
        <Select
            isOpen={isOpen}
            selected={currentSelection}
            onSelect={onSelect}
            onOpenChange={setIsOpen}
            toggle={toggle}
            popperProps={{
                position: 'right',
            }}
        >
            <SelectList style={{ maxHeight: '50vh', overflow: 'auto' }}>
                <div className="pf-v5-u-p-md">
                    <SearchInput
                        value={filterValue}
                        onChange={(_event, value) => setFilterValue(value)}
                        placeholder="Filter by cluster"
                        aria-label="Filter clusters"
                    />
                </div>
                <Divider />
                <SelectOption
                    key={SELECT_ALL}
                    value={SELECT_ALL}
                    hasCheckbox
                    isSelected={currentSelection === SELECT_ALL}
                >
                    All clusters
                </SelectOption>
                {filteredClusters.length > 0 && (
                    <Divider className="pf-v5-u-mb-0" component="div" />
                )}
                {filteredClusters.map(({ name }) => (
                    <SelectOption
                        key={name}
                        value={name}
                        hasCheckbox
                        isSelected={
                            currentSelection !== SELECT_ALL && currentSelection.includes(name)
                        }
                    >
                        {name}
                    </SelectOption>
                ))}
            </SelectList>
        </Select>
    );
}

export default ClusterSelect;
