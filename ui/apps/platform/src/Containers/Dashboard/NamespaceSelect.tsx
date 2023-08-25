import React, { ChangeEvent } from 'react';
import {
    Divider,
    Select,
    SelectGroup,
    SelectOption,
    SelectOptionObject,
    SelectVariant,
} from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { flattenFilterValue } from 'utils/searchUtils';
import { Cluster } from './types';

import './NamespaceSelect.css';

const selectAll: unique symbol = Symbol('Select-All-Namespaces');

function createOptions(clusters: Cluster[], filterValue?: string) {
    let clustersToShow: Pick<Cluster, 'name' | 'namespaces'>[] = [];

    if (filterValue) {
        clusters.forEach(({ name: clusterName, namespaces }) => {
            // If the search filter matches the name of the cluster, include all that cluster's
            // namespaces.
            if (clusterName.toLowerCase().includes(filterValue)) {
                clustersToShow.push({ name: clusterName, namespaces });
            } else {
                const namespacesToShow = namespaces.filter(({ metadata: { name } }) => {
                    return name.toLowerCase().includes(filterValue);
                });

                if (namespacesToShow.length > 0) {
                    clustersToShow.push({ name: clusterName, namespaces: namespacesToShow });
                }
            }
        });
    } else {
        clustersToShow = clusters;
    }

    return [
        <SelectOption key={selectAll.toString()} value={selectAll}>
            <span>All namespaces</span>
        </SelectOption>,
        <Divider key="namespace-select-option-divider" className="pf-u-mb-0" component="div" />,
        ...clustersToShow.map(({ name: clusterName, namespaces }) => (
            <SelectGroup key={clusterName} label={clusterName}>
                {namespaces.map(({ metadata: { id, name } }) => (
                    <SelectOption key={id} value={id}>
                        <span>{name}</span>
                    </SelectOption>
                ))}
            </SelectGroup>
        )),
    ];
}

type NamespaceSelectProps = {
    clusters: Cluster[];
    namespaceSearch: string | string[] | undefined;
    isDisabled?: boolean;
    onChange: (newClusterSearch: string[]) => void;
    onSelectAll: () => void;
};

function NamespaceSelect({
    clusters,
    namespaceSearch,
    isDisabled = false,
    onChange,
    onSelectAll,
}: NamespaceSelectProps) {
    const { isOpen, onToggle } = useSelectToggle();
    const currentSelection = flattenFilterValue(namespaceSearch, selectAll);
    const options = createOptions(clusters);

    function onSelect(e, selectedTarget: string | SelectOptionObject) {
        if (typeof selectedTarget !== 'string') {
            onSelectAll();
        } else if (currentSelection === selectAll) {
            onChange([selectedTarget]);
        } else {
            const newSelection = currentSelection.find((ns) => ns === selectedTarget)
                ? currentSelection.filter((ns) => ns !== selectedTarget)
                : currentSelection.concat(selectedTarget);
            onChange(newSelection);
        }
    }

    function onFilter(e: ChangeEvent<HTMLInputElement> | null, filterValue: string) {
        return createOptions(clusters, filterValue);
    }

    return (
        <Select
            toggleAriaLabel="Select namespaces"
            className="namespace-select"
            variant={SelectVariant.checkbox}
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            onFilter={onFilter}
            placeholderText={currentSelection === selectAll ? 'All namespaces' : 'Namespaces'}
            selections={currentSelection}
            isDisabled={isDisabled}
            maxHeight="50vh"
            position="right"
            width={210}
            isGrouped
            hasInlineFilter
        >
            {options}
        </Select>
    );
}

export default NamespaceSelect;
