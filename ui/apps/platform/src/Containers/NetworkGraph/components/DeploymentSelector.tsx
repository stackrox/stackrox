import React, { useCallback } from 'react';
import { Button, Select, SelectGroup, SelectOption, SelectVariant } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
import { DeploymentIcon } from '../common/NetworkGraphIcons';

type DeploymentSelectorProps = {
    deploymentsByNamespace: NamespaceWithDeployments[];
    selectedDeployments: string[];
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function DeploymentSelector({
    deploymentsByNamespace = [],
    selectedDeployments = [],
    searchFilter,
    setSearchFilter,
}: DeploymentSelectorProps) {
    const {
        isOpen: isDeploymentOpen,
        toggleSelect: toggleIsDeploymentOpen,
        closeSelect,
    } = useSelectToggle();

    const onFilterDeployments = useCallback(
        (_, filterValue: string) => {
            const filteredNamespaceDeployments = deploymentsByNamespace.map((namespace) => (
                <SelectGroup label={namespace.metadata.name} key={namespace.metadata.id}>
                    {namespace.deployments
                        .filter((deployment) =>
                            deployment.name.toLowerCase().includes(filterValue.toLowerCase())
                        )
                        .map((deployment) => (
                            <SelectOption key={deployment.id} value={deployment.name}>
                                <span>
                                    <DeploymentIcon />
                                    <span className="pf-u-mx-xs" data-testid="deployment-name">
                                        {deployment.name}
                                    </span>
                                </span>
                            </SelectOption>
                        ))}
                </SelectGroup>
            ));

            return filteredNamespaceDeployments;
        },
        [deploymentsByNamespace]
    );

    const onDeploymentSelect = (_, selected) => {
        const newSelection = selectedDeployments.find((nsFilter) => nsFilter === selected)
            ? selectedDeployments.filter((nsFilter) => nsFilter !== selected)
            : selectedDeployments.concat(selected);

        const modifiedSearchObject = { ...searchFilter };
        modifiedSearchObject.Deployment = newSelection;
        setSearchFilter(modifiedSearchObject);
    };

    const onClearSelections = () => {
        const modifiedSearchObject = { ...searchFilter };
        delete modifiedSearchObject.Deployment;
        closeSelect();
        setSearchFilter(modifiedSearchObject);
    };

    const deploymentSelectOptions: JSX.Element[] = deploymentsByNamespace.map((namespace) => (
        <SelectGroup label={namespace.metadata.name} key={namespace.metadata.id}>
            {namespace.deployments.map((deployment) => (
                <SelectOption key={deployment.id} value={deployment.name}>
                    <span>
                        <DeploymentIcon /> {deployment.name}
                    </span>
                </SelectOption>
            ))}
        </SelectGroup>
    ));

    return (
        <Select
            isOpen={isDeploymentOpen}
            onToggle={toggleIsDeploymentOpen}
            onSelect={onDeploymentSelect}
            onFilter={onFilterDeployments}
            className="deployment-select"
            placeholderText={
                <span>
                    <DeploymentIcon className="pf-u-mr-xs" />{' '}
                    <span style={{ position: 'relative', top: '1px' }}>Deployments</span>
                </span>
            }
            toggleAriaLabel="Select deployments"
            isDisabled={deploymentsByNamespace.length === 0}
            selections={selectedDeployments}
            variant={SelectVariant.checkbox}
            maxHeight="275px"
            hasInlineFilter
            isGrouped
            isPlain
            footer={
                <Button variant="link" isInline onClick={onClearSelections}>
                    Clear selections
                </Button>
            }
        >
            {deploymentSelectOptions}
        </Select>
    );
}

export default DeploymentSelector;
