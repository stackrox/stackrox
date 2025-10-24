import { useState } from 'react';
import type { MouseEvent, Ref } from 'react';
import {
    Badge,
    Divider,
    Flex,
    FlexItem,
    MenuToggle,
    SearchInput,
    Select,
    SelectGroup,
    SelectList,
    SelectOption,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import { flattenFilterValue } from 'utils/searchUtils';
import type { Cluster } from './types';

import './NamespaceSelect.css';

// TODO: Refactor ClusterSelect and NamespaceSelect to use a shared reusable component
const SELECT_ALL = '##SELECT_ALL##';

type NamespaceSelectProps = {
    clusters: Cluster[];
    namespaceSearch: string | string[] | undefined;
    isDisabled?: boolean;
    onChange: (newClusterSearch: string[]) => void;
    onSelectAll: () => void;
};

function NamespaceSelect({
    clusters,
    namespaceSearch,
    isDisabled = false,
    onChange,
    onSelectAll,
}: NamespaceSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [filterValue, setFilterValue] = useState('');
    const currentSelection = flattenFilterValue(namespaceSearch, SELECT_ALL);

    const onToggle = () => {
        const willBeOpen = !isOpen;
        setIsOpen(willBeOpen);
        if (!willBeOpen) {
            setFilterValue('');
        }
    };

    function onSelect(_event: MouseEvent | undefined, selectedTarget: string | number | undefined) {
        if (selectedTarget === SELECT_ALL) {
            onSelectAll();
        } else if (typeof selectedTarget !== 'string') {
            // Do nothing for invalid types
        } else if (currentSelection === SELECT_ALL) {
            onChange([selectedTarget]);
        } else {
            const newSelection = currentSelection.includes(selectedTarget)
                ? currentSelection.filter((ns) => ns !== selectedTarget)
                : currentSelection.concat(selectedTarget);
            onChange(newSelection);
        }
    }

    const filteredClusters = filterValue
        ? clusters
              .map(({ name: clusterName, namespaces }) => {
                  const namespacesToShow = namespaces.filter(({ metadata: { name } }) => {
                      return name.toLowerCase().includes(filterValue.toLowerCase());
                  });

                  if (namespacesToShow.length > 0) {
                      return { name: clusterName, namespaces: namespacesToShow };
                  }
                  return null;
              })
              .filter(
                  (cluster): cluster is Pick<Cluster, 'name' | 'namespaces'> => cluster !== null
              )
        : clusters;

    const toggle = (toggleRef: Ref<MenuToggleElement>) => {
        const numSelected = currentSelection === SELECT_ALL ? 0 : currentSelection.length;
        const placeholderText = currentSelection === SELECT_ALL ? 'All namespaces' : 'Namespaces';

        return (
            <MenuToggle
                ref={toggleRef}
                onClick={onToggle}
                isExpanded={isOpen}
                isDisabled={isDisabled}
                style={{ width: '210px' }}
                aria-label="Select namespaces"
            >
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                >
                    <FlexItem>{placeholderText}</FlexItem>
                    {numSelected > 0 && <Badge isRead>{numSelected}</Badge>}
                </Flex>
            </MenuToggle>
        );
    };

    return (
        <Select
            className="namespace-select"
            isOpen={isOpen}
            selected={currentSelection}
            onSelect={onSelect}
            onOpenChange={setIsOpen}
            toggle={toggle}
            popperProps={{
                position: 'right',
            }}
        >
            <SelectList style={{ maxHeight: '50vh', overflow: 'auto' }}>
                <div className="pf-v5-u-p-md">
                    <SearchInput
                        value={filterValue}
                        onChange={(_event, value) => setFilterValue(value)}
                        placeholder="Filter by namespace"
                        aria-label="Filter namespaces"
                    />
                </div>
                <Divider />
                <SelectOption
                    key={SELECT_ALL}
                    value={SELECT_ALL}
                    hasCheckbox
                    isSelected={currentSelection === SELECT_ALL}
                >
                    All namespaces
                </SelectOption>
                {filteredClusters.length > 0 && (
                    <Divider className="pf-v5-u-mb-0" component="div" />
                )}
                {filteredClusters.map(({ name: clusterName, namespaces }) => (
                    <SelectGroup key={clusterName} label={clusterName}>
                        {namespaces.map(({ metadata: { id, name } }) => (
                            <SelectOption
                                key={id}
                                value={id}
                                hasCheckbox
                                isSelected={
                                    currentSelection !== SELECT_ALL && currentSelection.includes(id)
                                }
                            >
                                {name}
                            </SelectOption>
                        ))}
                    </SelectGroup>
                ))}
            </SelectList>
        </Select>
    );
}

export default NamespaceSelect;
