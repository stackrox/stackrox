import React, { useCallback, ChangeEvent } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Select,
    SelectOption,
    SelectVariant,
} from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import useURLSearch from 'hooks/useURLSearch';
import useFetchClusterNamespaces from 'hooks/useFetchClusterNamespaces';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { ClusterIcon, NamespaceIcon } from '../common/NetworkGraphIcons';
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

type NetworkBreadcrumbsProps = {
    clusters: Cluster[];
};

function NetworkBreadcrumbs({ clusters = [] }: NetworkBreadcrumbsProps) {
    const {
        isOpen: isClusterOpen,
        toggleSelect: toggleIsClusterOpen,
        closeSelect: closeClusterSelect,
    } = useSelectToggle();
    const {
        isOpen: isNamespaceOpen,
        toggleSelect: toggleIsNamespaceOpen,
        closeSelect: closeNamespaceSelect,
    } = useSelectToggle();

    const { searchFilter, setSearchFilter } = useURLSearch();

    const {
        cluster: selectedClusterName,
        namespaces: selectedNamespaces,
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        deployments: deploymentsFromUrl,
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        remainingQuery,
    } = getScopeHierarchy(searchFilter);
    const selectedClusterId =
        clusters.find((cluster) => cluster.name === selectedClusterName)?.id || '';
    const { loading, error, namespaces } = useFetchClusterNamespaces(selectedClusterId);

    const onClusterSelect = (_, value) => {
        closeClusterSelect();

        if (value !== selectedClusterName) {
            const modifiedSearchObject = { ...searchFilter };
            modifiedSearchObject.Cluster = value;
            setSearchFilter(modifiedSearchObject);
        }
    };

    const onFilterNamespaces = useCallback(
        (e: ChangeEvent<HTMLInputElement> | null, filterValue: string) =>
            filterElementsWithValueProp(
                filterValue,
                namespaces.map((filteredNS) => (
                    <SelectOption key={filteredNS} value={filteredNS}>
                        <span>
                            <NamespaceIcon /> {filteredNS}
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

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => (
        <SelectOption key={cluster.id} value={cluster.name}>
            <span>
                <ClusterIcon /> {cluster.name}
            </span>
        </SelectOption>
    ));
    const namespaceSelectOptions: JSX.Element[] = namespaces.map((namespace) => (
        <SelectOption key={namespace} value={namespace}>
            <span>
                <NamespaceIcon /> {namespace}
            </span>
        </SelectOption>
    ));

    return (
        <>
            <Breadcrumb>
                <BreadcrumbItem isDropdown>
                    <Select
                        isPlain
                        placeholderText={<em>Select a cluster</em>}
                        aria-label="Select a cluster"
                        onToggle={toggleIsClusterOpen}
                        onSelect={onClusterSelect}
                        isOpen={isClusterOpen}
                        selections={selectedClusterName}
                    >
                        {clusterSelectOptions}
                    </Select>
                </BreadcrumbItem>
                <BreadcrumbItem isDropdown>
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
                        {/* {availableNamespaceFilters.map((nsFilter) => (
                            <SelectOption key={nsFilter} value={nsFilter} />
                        ))} */}
                    </Select>
                </BreadcrumbItem>
            </Breadcrumb>
        </>
    );
}

export default NetworkBreadcrumbs;
