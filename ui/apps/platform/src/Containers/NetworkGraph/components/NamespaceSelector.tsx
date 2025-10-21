import { useMemo, useState } from 'react';
import type { ReactElement, Ref } from 'react';
import {
    Badge,
    Button,
    Divider,
    Flex,
    FlexItem,
    Menu,
    MenuContent,
    MenuFooter,
    MenuSearch,
    MenuItem,
    MenuList,
    SearchInput,
    MenuSearchInput,
    Select,
    MenuToggle,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import type { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
import type { NamespaceScopeObject } from 'services/RolesService';
import { NamespaceIcon } from '../common/NetworkGraphIcons';

export function getDeploymentLookupMap(
    deploymentsByNamespace: NamespaceWithDeployments[]
): Record<string, string[]> {
    return deploymentsByNamespace.reduce<Record<string, string[]>>((acc, ns) => {
        const deployments = ns.deployments.map((deployment) => deployment.name);
        return { ...acc, [ns.metadata.name]: deployments };
    }, {});
}

export function getDeploymentsAllowedByNamespaces(
    deploymentLookupMap: Record<string, string[]>,
    namespaceSelection: string[]
) {
    const newDeploymentLookup = Object.fromEntries(
        Object.entries(deploymentLookupMap).filter(([key]) => namespaceSelection.includes(key))
    );
    const allowedDeployments = Object.values(newDeploymentLookup).flat(1);

    return allowedDeployments;
}

type NamespaceSelectorProps = {
    namespaces?: NamespaceScopeObject[];
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
}: NamespaceSelectorProps): ReactElement {
    const { isOpen: isNamespaceOpen, toggleSelect: toggleIsNamespaceOpen } = useSelectToggle();
    const [input, setInput] = useState('');

    const handleTextInputChange = (value: string) => {
        setInput(value);
    };

    const clusterSelected = Boolean(searchFilter?.Cluster);
    const isEmptyCluster = clusterSelected && namespaces.length === 0;

    const deploymentLookupMap = getDeploymentLookupMap(deploymentsByNamespace);

    const filteredDeploymentSelectMenuItems = useMemo(() => {
        const namespaceSelectMenuItems = namespaces
            .filter((namespace) =>
                namespace.name.toLowerCase().includes(input.toString().toLowerCase())
            )
            .map((namespace) => {
                return (
                    <MenuItem
                        key={namespace.id}
                        hasCheckbox
                        itemId={namespace.name}
                        isSelected={selectedNamespaces.includes(namespace.name)}
                    >
                        <span>
                            <NamespaceIcon />
                            <span className="pf-v5-u-mx-xs" data-testid="namespace-name">
                                {namespace.name}
                            </span>
                        </span>
                    </MenuItem>
                );
            });

        return namespaceSelectMenuItems;
    }, [namespaces, input, selectedNamespaces]);

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
        setSearchFilter(modifiedSearchObject);
    };

    const namespaceSelectMenu = (
        <Menu onSelect={onNamespaceSelect} selected={selectedNamespaces}>
            <MenuSearch>
                <MenuSearchInput>
                    <SearchInput
                        value={input}
                        aria-label="Filter namespaces"
                        placeholder="Filter namespaces..."
                        onChange={(_event, value) => handleTextInputChange(value)}
                    />
                </MenuSearchInput>
            </MenuSearch>
            <Divider className="pf-v5-u-m-0" />
            <MenuContent>
                <MenuList className="network-graph-menu-list">
                    {filteredDeploymentSelectMenuItems.length === 0 && (
                        <MenuItem isDisabled key="no result">
                            No namespaces found
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
                    isDisabled={selectedNamespaces.length === 0}
                >
                    Clear selections
                </Button>
            </MenuFooter>
        </Menu>
    );

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={() => toggleIsNamespaceOpen(!isNamespaceOpen)}
            isExpanded={isNamespaceOpen}
            isDisabled={namespaces.length === 0}
            aria-label="Select namespaces"
            className="namespace-select"
            variant="plainText"
        >
            <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                <FlexItem
                    spacer={{ default: 'spacerSm' }}
                    alignSelf={{ default: 'alignSelfCenter' }}
                >
                    <NamespaceIcon />
                </FlexItem>
                <FlexItem spacer={{ default: 'spacerSm' }}>
                    <span style={{ position: 'relative', top: '1px' }}>
                        {isEmptyCluster ? 'No namespaces' : 'Namespaces'}
                    </span>
                </FlexItem>
                {selectedNamespaces.length !== 0 && (
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                        <Badge isRead>{selectedNamespaces.length}</Badge>
                    </FlexItem>
                )}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            isOpen={isNamespaceOpen}
            onOpenChange={(nextOpen: boolean) => toggleIsNamespaceOpen(nextOpen)}
            toggle={toggle}
            popperProps={{
                maxWidth: '400px',
                direction: 'down',
            }}
        >
            {namespaceSelectMenu}
        </Select>
    );
}

export default NamespaceSelector;
