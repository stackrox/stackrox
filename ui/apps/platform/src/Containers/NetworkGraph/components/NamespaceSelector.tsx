import React, { useCallback, ChangeEvent } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useFetchClusterNamespaces from 'hooks/useFetchClusterNamespaces';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { NamespaceIcon } from '../common/NetworkGraphIcons';
import getScopeHierarchy from '../utils/getScopeHierarchy';

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

type NamespaceSelectorProps = {
    selectedClusterId: string;
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function NamespaceSelector({
    selectedClusterId = '',
    searchFilter,
    setSearchFilter,
}: NamespaceSelectorProps) {
    const {
        isOpen: isNamespaceOpen,
        toggleSelect: toggleIsNamespaceOpen,
        closeSelect: closeNamespaceSelect,
    } = useSelectToggle();

    const { namespaces: selectedNamespaces } = getScopeHierarchy(searchFilter);
    const { loading, error, namespaces } = useFetchClusterNamespaces(selectedClusterId);

    const onFilterNamespaces = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                namespaces.map((namespace) => (
                    <SelectOption key={namespace} value={namespace}>
                        <span>
                            <NamespaceIcon /> {namespace}
                        </span>
                    </SelectOption>
                ))
            ),
        [namespaces]
    );

    const onNamespaceSelect = (_, selected) => {
        closeNamespaceSelect();

        const newSelection = selectedNamespaces.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaces.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaces.concat(selected);

        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Namespace = newSelection;
        setSearchFilter(modifiedSearchObject);
    };

    const namespaceSelectOptions: JSX.Element[] = namespaces.map((namespace) => (
        <SelectOption key={namespace} value={namespace}>
            <span>
                <NamespaceIcon /> {namespace}
            </span>
        </SelectOption>
    ));

    return (
        <Select
            isOpen={isNamespaceOpen}
            onToggle={toggleIsNamespaceOpen}
            onSelect={onNamespaceSelect}
            onFilter={onFilterNamespaces}
            className="namespace-select"
            placeholderText="Namespaces"
            isDisabled={!selectedClusterId || loading || Boolean(error)}
            selections={selectedNamespaces}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
        >
            {namespaceSelectOptions}
        </Select>
    );
}

export default NamespaceSelector;
