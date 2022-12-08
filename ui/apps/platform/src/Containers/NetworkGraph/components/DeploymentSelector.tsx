import React, { useCallback, ChangeEvent } from 'react';
import { Select, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
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
    deployments: string[];
    selectedDeployments: string[];
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function DeploymentSelector({
    deployments = [],
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
                deployments.map((deployment) => (
                    <SelectOption key={deployment} value={deployment}>
                        <span>
                            <DeploymentIcon /> {deployment}
                        </span>
                    </SelectOption>
                ))
            ),
        [deployments]
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

    const deploymentSelectOptions: JSX.Element[] = deployments.map((deployment) => (
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
            isDisabled={deployments.length === 0}
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
