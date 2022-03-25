import React, { useCallback, ChangeEvent } from 'react';
import { useDispatch } from 'react-redux';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import { actions as graphActions } from 'reducers/network/graph';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useNamespaceFilters from './useNamespaceFilters';

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
    id?: string;
    isDisabled?: boolean;
    className?: string;
}

function NamespaceSelect({ id, className = '', isDisabled = false }: NamespaceSelectProps) {
    const { isOpen, onToggle } = useSelectToggle();
    const { loading, error, availableNamespaceFilters, selectedNamespaceFilters } =
        useNamespaceFilters();
    const dispatch = useDispatch();

    function onSelect(e, selected) {
        const newSelection = selectedNamespaceFilters.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaceFilters.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaceFilters.concat(selected);
        dispatch(graphActions.setSelectedNamespaceFilters(newSelection));
    }

    const onFilter = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                availableNamespaceFilters.map((nsFilter) => (
                    <SelectOption key={nsFilter} value={nsFilter} />
                ))
            ),
        [availableNamespaceFilters]
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
            isDisabled={isDisabled || loading || Boolean(error)}
            selections={selectedNamespaceFilters}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
        >
            {availableNamespaceFilters.map((nsFilter) => (
                <SelectOption key={nsFilter} value={nsFilter} />
            ))}
        </Select>
    );
}

export default NamespaceSelect;
