import React, { useCallback, ChangeEvent } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { NamespaceWithDeployments } from 'hooks/useFetchClusterNamespaces';
import { DeploymentIcon } from '../common/NetworkGraphIcons';

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

type DeploymentSelectorProps = {
    deploymentsByNamespace: NamespaceWithDeployments[];
    selectedDeployments: string[];
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

/*
      <SelectGroup label="Status" key="group1">
        <SelectOption key={0} value="Running" />
        <SelectOption key={1} value="Stopped" />
        <SelectOption key={2} value="Down" />
        <SelectOption key={3} value="Degraded" />
        <SelectOption key={4} value="Needs maintenance" />
      </SelectGroup>,
      <SelectGroup label="Vendor names" key="group2">
        <SelectOption key={5} value="Dell" />
        <SelectOption key={6} value="Samsung" isDisabled />
        <SelectOption key={7} value="Hewlett-Packard" />
      </SelectGroup>

*/
function DeploymentSelector({
    deploymentsByNamespace = [],
    selectedDeployments = [],
    searchFilter,
    setSearchFilter,
}: DeploymentSelectorProps) {
    const {
        isOpen: isDeploymentOpen,
        toggleSelect: toggleIsDeploymentOpen,
        closeSelect: closeDeploymentSelect,
    } = useSelectToggle();

    const onFilterDeployments = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                deploymentsByNamespace.map((deployment) => (
                    <SelectOption key={deployment} value={deployment}>
                        <span>
                            <DeploymentIcon /> {deployment}
                        </span>
                    </SelectOption>
                ))
            ),
        [deploymentsByNamespace]
    );

    const onDeploymentSelect = (_, selected) => {
        closeDeploymentSelect();

        const newSelection = selectedDeployments.find((nsFilter) => nsFilter === selected)
            ? selectedDeployments.filter((nsFilter) => nsFilter !== selected)
            : selectedDeployments.concat(selected);

        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Deployment = newSelection;
        setSearchFilter(modifiedSearchObject);
    };

    const deploymentSelectOptions: JSX.Element[] = deploymentsByNamespace.map((deployment) => (
        <SelectOption key={deployment} value={deployment}>
            <span>
                <DeploymentIcon /> {deployment}
            </span>
        </SelectOption>
    ));

    return (
        <Select
            isOpen={isDeploymentOpen}
            onToggle={toggleIsDeploymentOpen}
            onSelect={onDeploymentSelect}
            onFilter={onFilterDeployments}
            className="deployment-select"
            placeholderText="Deployments"
            isDisabled={deploymentsByNamespace.length === 0}
            selections={selectedDeployments}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
        >
            {deploymentSelectOptions}
        </Select>
    );
}

export default DeploymentSelector;
