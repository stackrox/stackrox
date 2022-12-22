import React, { useCallback, ChangeEvent } from 'react';
import { Badge, Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Namespace } from 'hooks/useFetchClusterNamespaces';
import { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
import { NamespaceIcon } from '../common/NetworkGraphIcons';

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
    namespaces?: Namespace[];
    selectedNamespaces?: string[];
    selectedDeployments?: string[];
    deploymentsByNamespace?: NamespaceWithDeployments[];
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function NamespaceSelector({
    namespaces = [],
    selectedNamespaces = [],
    selectedDeployments = [],
    deploymentsByNamespace = [],
    searchFilter,
    setSearchFilter,
}: NamespaceSelectorProps) {
    const {
        isOpen: isNamespaceOpen,
        toggleSelect: toggleIsNamespaceOpen,
        closeSelect: closeNamespaceSelect,
    } = useSelectToggle();

    const onFilterNamespaces = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                namespaces.map((namespace) => (
                    <SelectOption
                        key={namespace.metadata.id}
                        value={namespace.metadata.name}
                        isDisabled={namespace.deploymentCount < 1}
                    >
                        <span>
                            <NamespaceIcon /> {namespace.metadata.name}{' '}
                            <Badge isRead>{namespace.deploymentCount}</Badge>
                        </span>
                    </SelectOption>
                ))
            ),
        [namespaces]
    );

    const deploymentLookup: Record<string, string[]> = deploymentsByNamespace.reduce((acc, ns) => {
        const deployments = ns.deployments.map((deployment) => deployment.name);
        return { ...acc, [ns.metadata.name]: deployments };
    }, {});

    const onNamespaceSelect = (_, selected) => {
        closeNamespaceSelect();

        const newSelection = selectedNamespaces.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaces.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaces.concat(selected);

        const newDeploymentLookup = Object.fromEntries(
            Object.entries(deploymentLookup).filter(([key]) => newSelection.includes(key))
        );
        const allowedDeployments = Object.values(newDeploymentLookup).flat(1);

        const filteredSelectedDeployments = selectedDeployments.filter((deployment) =>
            allowedDeployments.includes(deployment)
        );

        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Namespace = newSelection;
        modifiedSearchObject.Deployment = filteredSelectedDeployments;
        setSearchFilter(modifiedSearchObject);
    };

    const namespaceSelectOptions: JSX.Element[] = namespaces.map((namespace) => {
        return (
            <SelectOption
                key={namespace.metadata.id}
                value={namespace.metadata.name}
                isDisabled={namespace.deploymentCount < 1}
            >
                <span>
                    <NamespaceIcon /> {namespace.metadata.name}{' '}
                    <Badge isRead>{namespace.deploymentCount}</Badge>
                </span>
            </SelectOption>
        );
    });

    return (
        <Select
            isOpen={isNamespaceOpen}
            onToggle={toggleIsNamespaceOpen}
            onSelect={onNamespaceSelect}
            onFilter={onFilterNamespaces}
            className="namespace-select"
            placeholderText="Namespaces"
            isDisabled={namespaceSelectOptions.length === 0}
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
