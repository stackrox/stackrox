import React, { useCallback, ChangeEvent } from 'react';
import { Button, Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Namespace } from 'hooks/useFetchClusterNamespacesForPermissions';
import { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
import { NamespaceIcon } from '../common/NetworkGraphIcons';
import { getDeploymentLookupMap, getDeploymentsAllowedByNamespaces } from '../utils/hierarchyUtils';

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
        closeSelect,
    } = useSelectToggle();

    const onFilterNamespaces = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                namespaces.map((namespace) => (
                    <SelectOption key={namespace.id} value={namespace.name}>
                        <span>
                            <NamespaceIcon />
                            <span className="pf-u-mx-xs" data-testid="namespace-name">
                                {namespace.name}
                            </span>
                        </span>
                    </SelectOption>
                ))
            ),
        [namespaces]
    );

    const clusterSelected = Boolean(searchFilter?.Cluster);
    const isEmptyCluster = clusterSelected && namespaces.length === 0;

    const deploymentLookupMap = getDeploymentLookupMap(deploymentsByNamespace);

    const onNamespaceSelect = (_, selected) => {
        const newSelection = selectedNamespaces.find((nsFilter) => nsFilter === selected)
            ? selectedNamespaces.filter((nsFilter) => nsFilter !== selected)
            : selectedNamespaces.concat(selected);

        const allowedDeployments = getDeploymentsAllowedByNamespaces(
            deploymentLookupMap,
            newSelection
        );

        const filteredSelectedDeployments = selectedDeployments.filter((deployment) =>
            allowedDeployments.includes(deployment)
        );

        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Namespace = newSelection;
        modifiedSearchObject.Deployment = filteredSelectedDeployments;
        setSearchFilter(modifiedSearchObject);
    };

    const onClearSelections = () => {
        const modifiedSearchObject = { ...searchFilter };
        delete modifiedSearchObject.Namespace;
        delete modifiedSearchObject.Deployment;
        closeSelect();
        setSearchFilter(modifiedSearchObject);
    };

    const namespaceSelectOptions: JSX.Element[] = namespaces.map((namespace) => {
        return (
            <SelectOption key={namespace.id} value={namespace.name}>
                <span>
                    <NamespaceIcon />
                    <span className="pf-u-mx-xs" data-testid="namespace-name">
                        {namespace.name}
                    </span>
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
            placeholderText={
                <span>
                    <NamespaceIcon className="pf-u-mr-xs" />{' '}
                    <span style={{ position: 'relative', top: '1px' }}>
                        {isEmptyCluster ? 'No namespaces' : 'Namespaces'}
                    </span>
                </span>
            }
            toggleAriaLabel="Select namespaces"
            isDisabled={namespaceSelectOptions.length === 0}
            selections={selectedNamespaces}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
            isPlain
            footer={
                <Button variant="link" isInline onClick={onClearSelections}>
                    Clear selections
                </Button>
            }
        >
            {namespaceSelectOptions}
        </Select>
    );
}

export default NamespaceSelector;
