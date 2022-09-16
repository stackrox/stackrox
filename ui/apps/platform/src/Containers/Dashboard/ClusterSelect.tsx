import React, { ChangeEvent } from 'react';
import {
    Divider,
    Select,
    SelectOption,
    SelectOptionObject,
    SelectVariant,
} from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { flattenFilterValue } from 'utils/searchUtils';
import { Cluster } from './types';

const selectAll: unique symbol = Symbol('Select-All-Clusters');

function createOptions(clusters: Cluster[], filterValue?: string) {
    const visibleClusters = filterValue
        ? clusters.filter(({ name }) => name.toLowerCase().includes(filterValue.toLowerCase()))
        : clusters;

    return [
        <SelectOption key={selectAll.toString()} value={selectAll}>
            <span>All clusters</span>
        </SelectOption>,
        <Divider key="cluster-select-option-divider" className="pf-u-mb-0" component="div" />,
        ...visibleClusters.map(({ name }) => (
            <SelectOption key={name} value={name}>
                <span>{name}</span>
            </SelectOption>
        )),
    ];
}

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
    const { isOpen, onToggle } = useSelectToggle();
    const currentSelection = flattenFilterValue(clusterSearch, selectAll);
    const options = createOptions(clusters);

    function onSelect(e, selectedTarget: string | SelectOptionObject) {
        if (typeof selectedTarget !== 'string') {
            onSelectAll();
        } else if (currentSelection === selectAll) {
            onChange({ type: 'add', value: selectedTarget, selection: [selectedTarget] });
        } else {
            const isRemoval = Boolean(currentSelection.find((cs) => cs === selectedTarget));
            const selection = isRemoval
                ? currentSelection.filter((cs) => cs !== selectedTarget)
                : currentSelection.concat(selectedTarget);

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

    function onFilter(e: ChangeEvent<HTMLInputElement> | null, filterValue: string) {
        return createOptions(clusters, filterValue);
    }

    return (
        <Select
            toggleAriaLabel="Select clusters"
            variant={SelectVariant.checkbox}
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            onFilter={onFilter}
            placeholderText={currentSelection === selectAll ? 'All clusters' : 'Clusters'}
            selections={currentSelection}
            isDisabled={isDisabled}
            maxHeight="50vh"
            width={180}
            position="right"
            hasInlineFilter
        >
            {options}
        </Select>
    );
}

export default ClusterSelect;
