import React, { useMemo } from 'react';
import {
    Badge,
    Button,
    Divider,
    Flex,
    FlexItem,
    Menu,
    MenuContent,
    MenuFooter,
    MenuGroup,
    MenuInput,
    MenuItem,
    MenuList,
    SearchInput,
    Select,
} from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
import { removeNullValues } from 'utils/removeNullValues';
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
    const { isOpen: isDeploymentOpen, toggleSelect: toggleIsDeploymentOpen } = useSelectToggle();
    const [input, setInput] = React.useState('');

    const handleTextInputChange = (value: string) => {
        setInput(value);
    };

    const filteredDeploymentSelectMenuItems = useMemo(() => {
        let deploymentSelectMenuItems = deploymentsByNamespace.map((namespace) => {
            const menuItems = namespace.deployments
                .filter((deployment) =>
                    deployment.name.toLowerCase().includes(input.toString().toLowerCase())
                )
                .map((deployment) => (
                    <MenuItem
                        key={deployment.id}
                        hasCheck
                        itemId={deployment.name}
                        isSelected={selectedDeployments.includes(deployment.name)}
                    >
                        <span>
                            <DeploymentIcon />
                            <span className="pf-u-mx-xs" data-testid="deployment-name">
                                {deployment.name}
                            </span>
                        </span>
                    </MenuItem>
                ));
            if (menuItems.length === 0) {
                return null;
            }
            return (
                <MenuGroup
                    key={namespace.metadata.name}
                    label={namespace.metadata.name}
                    labelHeadingLevel="h3"
                >
                    <MenuList>{menuItems}</MenuList>
                </MenuGroup>
            );
        });
        deploymentSelectMenuItems = removeNullValues(deploymentSelectMenuItems);
        return deploymentSelectMenuItems;
    }, [deploymentsByNamespace, input, selectedDeployments]);

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
        setSearchFilter(modifiedSearchObject);
    };

    const deploymentSelectMenu = (
        <Menu onSelect={onDeploymentSelect} selected={selectedDeployments} isScrollable>
            <MenuInput className="pf-u-p-md">
                <SearchInput
                    value={input}
                    aria-label="Filter deployments"
                    type="search"
                    placeholder="Filter deployments..."
                    onChange={(_event, value) => handleTextInputChange(value)}
                />
            </MenuInput>
            <Divider className="pf-u-m-0" />
            <MenuContent>
                <MenuList>
                    {filteredDeploymentSelectMenuItems.length === 0 && (
                        <MenuItem isDisabled key="no result">
                            No deployments found
                        </MenuItem>
                    )}
                    {filteredDeploymentSelectMenuItems}
                </MenuList>
            </MenuContent>
            <MenuFooter>
                <Button
                    variant="link"
                    isInline
                    onClick={onClearSelections}
                    isDisabled={selectedDeployments.length === 0}
                >
                    Clear selections
                </Button>
            </MenuFooter>
        </Menu>
    );

    return (
        <Select
            isOpen={isDeploymentOpen}
            onToggle={toggleIsDeploymentOpen}
            className="deployment-select"
            placeholderText={
                <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                    <FlexItem
                        spacer={{ default: 'spacerSm' }}
                        alignSelf={{ default: 'alignSelfCenter' }}
                    >
                        <DeploymentIcon />
                    </FlexItem>
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                        <span style={{ position: 'relative', top: '1px' }}>Deployments</span>
                    </FlexItem>
                    {selectedDeployments.length !== 0 && (
                        <FlexItem spacer={{ default: 'spacerSm' }}>
                            <Badge isRead>{selectedDeployments.length}</Badge>
                        </FlexItem>
                    )}
                </Flex>
            }
            toggleAriaLabel="Select deployments"
            isDisabled={deploymentsByNamespace.length === 0}
            isPlain
            customContent={deploymentSelectMenu}
        />
    );
}

export default DeploymentSelector;
