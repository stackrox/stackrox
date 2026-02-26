import { useMemo, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, ReactElement, Ref } from 'react';
import {
    Badge,
    Button,
    Divider,
    Flex,
    FlexItem,
    MenuFooter,
    MenuSearch,
    MenuSearchInput,
    MenuToggle,
    SearchInput,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import type { NamespaceWithDeployments } from 'hooks/useFetchNamespaceDeployments';
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
}: DeploymentSelectorProps): ReactElement {
    const { isOpen: isDeploymentOpen, toggleSelect: toggleIsDeploymentOpen } = useSelectToggle();
    const [input, setInput] = useState('');

    const handleTextInputChange = (value: string) => {
        setInput(value);
    };

    const filteredDeploymentSelectOptions = useMemo(() => {
        const groups = deploymentsByNamespace
            .map((namespace) => {
                const options = namespace.deployments
                    .filter((deployment) =>
                        deployment.name.toLowerCase().includes(input.toLowerCase())
                    )
                    .map((deployment) => (
                        <SelectOption
                            key={deployment.id}
                            hasCheckbox
                            value={deployment.name}
                            isSelected={selectedDeployments.includes(deployment.name)}
                        >
                            <span>
                                <DeploymentIcon />
                                <span className="pf-v6-u-mx-xs" data-testid="deployment-name">
                                    {deployment.name}
                                </span>
                            </span>
                        </SelectOption>
                    ));
                if (options.length === 0) {
                    return null;
                }
                return (
                    <SelectGroup key={namespace.metadata.name} label={namespace.metadata.name}>
                        {options}
                    </SelectGroup>
                );
            })
            .filter((group): group is JSX.Element => group !== null);
        return groups;
    }, [deploymentsByNamespace, input, selectedDeployments]);

    const onDeploymentSelect = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        selected: string | number | undefined
    ) => {
        if (typeof selected !== 'string') {
            return;
        }
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

    const toggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            onClick={() => toggleIsDeploymentOpen(!isDeploymentOpen)}
            isExpanded={isDeploymentOpen}
            isDisabled={deploymentsByNamespace.length === 0}
            aria-label="Select deployments"
            className="deployment-select"
            variant="plainText"
        >
            <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                <FlexItem
                    spacer={{ default: 'spacerSm' }}
                    alignSelf={{ default: 'alignSelfCenter' }}
                >
                    <DeploymentIcon />
                </FlexItem>
                <FlexItem spacer={{ default: 'spacerSm' }}>Deployments</FlexItem>
                {selectedDeployments.length !== 0 && (
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                        <Badge isRead>{selectedDeployments.length}</Badge>
                    </FlexItem>
                )}
            </Flex>
        </MenuToggle>
    );

    return (
        <Select
            isOpen={isDeploymentOpen}
            onOpenChange={(nextOpen: boolean) => toggleIsDeploymentOpen(nextOpen)}
            onSelect={onDeploymentSelect}
            selected={selectedDeployments}
            toggle={toggle}
            popperProps={{
                maxWidth: '400px',
                direction: 'down',
            }}
        >
            <MenuSearch>
                <MenuSearchInput>
                    <SearchInput
                        value={input}
                        aria-label="Filter deployments"
                        placeholder="Filter deployments..."
                        onChange={(_event, value) => handleTextInputChange(value)}
                    />
                </MenuSearchInput>
            </MenuSearch>
            <Divider className="pf-v6-u-m-0" />
            <SelectList className="network-graph-menu-list">
                {filteredDeploymentSelectOptions.length === 0 && (
                    <SelectOption isDisabled key="no result">
                        No deployments found
                    </SelectOption>
                )}
                {filteredDeploymentSelectOptions}
            </SelectList>
            <Divider />
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
        </Select>
    );
}

export default DeploymentSelector;
