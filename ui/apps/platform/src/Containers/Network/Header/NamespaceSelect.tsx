import React, { useCallback, ChangeEvent, useEffect } from 'react';
import { useDispatch } from 'react-redux';
import { Select, SelectOption, SelectOptionObject, SelectVariant } from '@patternfly/react-core';

import { actions as graphActions } from 'reducers/network/graph';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useURLNamespaceIds from 'hooks/useURLNamespaceIds';
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
    const { namespaceIds, setNamespaceIds } = useURLNamespaceIds(
        selectedNamespaceFilters,
        availableNamespaceFilters.map((ns) => ns.id)
    );
    const dispatch = useDispatch();

    useEffect(() => {
        dispatch(graphActions.setSelectedNamespaceFilters(namespaceIds));
    }, [dispatch, namespaceIds]);

    function onSelect(e, selected: string | SelectOptionObject) {
        const selectedString = typeof selected === 'string' ? selected : selected.toString();
        const newSelection = selectedNamespaceFilters.find(
            (nsFilter) => nsFilter === selectedString
        )
            ? selectedNamespaceFilters.filter((nsFilter) => nsFilter !== selectedString)
            : selectedNamespaceFilters.concat(selectedString);

        const cleanedSelection = newSelection
            .filter((ns) => availableNamespaceFilters.find((nsFilter) => nsFilter.id === ns))
            .map((ns) => ns);

        setNamespaceIds(cleanedSelection);
        dispatch(graphActions.setSelectedNamespaceFilters(cleanedSelection));
    }

    const onFilter = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                availableNamespaceFilters.map((nsFilter) => (
                    <SelectOption key={nsFilter.id} value={nsFilter.id}>
                        {nsFilter.name}
                    </SelectOption>
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
            selections={namespaceIds}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
        >
            {availableNamespaceFilters.map((nsFilter) => (
                <SelectOption key={nsFilter.id} value={nsFilter.id}>
                    {nsFilter.name}
                </SelectOption>
            ))}
        </Select>
    );
}

export default NamespaceSelect;
