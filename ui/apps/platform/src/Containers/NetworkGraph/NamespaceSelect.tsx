import React, { useCallback, ChangeEvent } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
// import useNamespaceFilters from './useNamespaceFilters';

function filterElementsWithValueProp(
    filterValue: string,
    elements: React.ReactElement[] | undefined
): React.ReactElement[] | undefined {
    if (filterValue === '' || elements === undefined) {
        return elements;
    }

    return elements.filter((reactElement) =>
        reactElement.props.value?.toLowerCase().includes(filterValue.toLowerCase())
    );
}
interface NamespaceSelectProps {
    namespaces: any[];
    id?: string;
    isDisabled?: boolean;
    className?: string;
    selectedNamespaces: string[];
    setSelectedNamespaces: (namespaces: string[]) => void;
}

function NamespaceSelect({
    namespaces = [],
    id = '',
    className = '',
    isDisabled = false,
    selectedNamespaces,
    setSelectedNamespaces,
}: NamespaceSelectProps) {
    const { isOpen, onToggle } = useSelectToggle();
    // const { loading, error, availableNamespaceFilters } = useNamespaceFilters(selectedClusterId);

    function onSelect(e, selected) {
        const newSelection = selectedNamespaces.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaces.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaces.concat(selected);
        setSelectedNamespaces(newSelection);
    }

    const onFilter = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                namespaces.map((nsFilter) => <SelectOption key={nsFilter} value={nsFilter} />)
            ),
        [namespaces]
    );

    // TODO Is there a more reliable way to set maxHeight here instead of hard coded px values?
    return (
        <Select
            id={id}
            isOpen={isOpen}
            onToggle={onToggle}
            onSelect={onSelect}
            onFilter={onFilter}
            className={`namespace-select ${className}`}
            placeholderText="Namespaces"
            isDisabled={isDisabled}
            selections={selectedNamespaces}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
        >
            {namespaces.map((nsFilter) => (
                <SelectOption key={nsFilter} value={nsFilter} />
            ))}
        </Select>
    );
}

export default NamespaceSelect;
