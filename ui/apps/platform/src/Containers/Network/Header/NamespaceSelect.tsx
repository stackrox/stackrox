import React, { useCallback, useRef, ChangeEvent, useEffect } from 'react';
import { useDispatch } from 'react-redux';
import isEqual from 'lodash/isEqual';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import { actions as graphActions } from 'reducers/network/graph';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useURLParameter from 'hooks/useURLParameter';
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

// TODO extract
// TODO Should we tie cluster in here as well? e.g. Multiple selected Clusters?
function useURLNamespaces(defaultNamespaces: string[], allowedNamespaces: string[]) {
    const [namespacesInternal, setNamespacesInternal] = useURLParameter(
        'ns',
        defaultNamespaces.filter((ns) => allowedNamespaces.includes(ns)) || []
    );
    const namespaceRef = useRef<string[]>([]);
    const setNamespaces = useCallback(
        (newNamespaces: string[]) => {
            setNamespacesInternal(newNamespaces.filter((ns) => allowedNamespaces.includes(ns)));
        },
        [setNamespacesInternal, allowedNamespaces]
    );

    const filteredNamespaces: string[] = [];
    if (!namespacesInternal) {
        // Do nothing
    } else if (
        typeof namespacesInternal === 'string' &&
        allowedNamespaces.includes(namespacesInternal)
    ) {
        filteredNamespaces.push(namespacesInternal);
    } else if (namespacesInternal && Array.isArray(namespacesInternal)) {
        namespacesInternal.forEach((ns) => {
            if (typeof ns === 'string' && allowedNamespaces.includes(ns)) {
                filteredNamespaces.push(ns);
            }
        });
    }

    if (!isEqual(namespaceRef.current, filteredNamespaces)) {
        namespaceRef.current = filteredNamespaces;
    }

    return {
        namespaces: namespaceRef.current,
        setNamespaces,
    };
}

function NamespaceSelect({ id, className = '', isDisabled = false }: NamespaceSelectProps) {
    const { isOpen, onToggle } = useSelectToggle();
    const { loading, error, availableNamespaceFilters, selectedNamespaceFilters } =
        useNamespaceFilters();
    const { namespaces, setNamespaces } = useURLNamespaces(
        selectedNamespaceFilters,
        availableNamespaceFilters
    );
    const dispatch = useDispatch();

    useEffect(() => {
        dispatch(graphActions.setSelectedNamespaceFilters(namespaces));
    }, [dispatch, namespaces]);

    function onSelect(e, selected) {
        const newSelection = selectedNamespaceFilters.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaceFilters.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaceFilters.concat(selected);

        const cleanedSelection = newSelection.filter((ns) =>
            availableNamespaceFilters.includes(ns)
        );

        setNamespaces(cleanedSelection);
        dispatch(graphActions.setSelectedNamespaceFilters(cleanedSelection));
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
            selections={namespaces}
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
