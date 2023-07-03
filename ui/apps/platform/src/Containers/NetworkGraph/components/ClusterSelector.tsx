import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { ClusterScopeObject } from 'services/RolesService';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { ClusterIcon } from '../common/NetworkGraphIcons';

export type ClusterSelectorProps = {
    clusters: ClusterScopeObject[];
    selectedClusterName?: string;
    searchFilter: Partial<Record<string, string | string[]>>;
    setSearchFilter: (newFilter: Partial<Record<string, string | string[]>>) => void;
};

function ClusterSelector({
    clusters = [],
    selectedClusterName = '',
    searchFilter,
    setSearchFilter,
}: ClusterSelectorProps) {
    const {
        isOpen: isClusterOpen,
        toggleSelect: toggleIsClusterOpen,
        closeSelect: closeClusterSelect,
    } = useSelectToggle();

    const onClusterSelect = (_, value) => {
        closeClusterSelect();

        if (value !== selectedClusterName) {
            const modifiedSearchObject = { ...searchFilter };
            modifiedSearchObject.Cluster = value;
            delete modifiedSearchObject.Namespace;
            delete modifiedSearchObject.Deployment;
            setSearchFilter(modifiedSearchObject);
        }
    };

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => (
        <SelectOption key={cluster.id} value={cluster.name}>
            <span>
                <ClusterIcon className="pf-u-mr-xs" /> {cluster.name}
            </span>
        </SelectOption>
    ));

    return (
        <Select
            className="cluster-select"
            isPlain
            placeholderText={
                <span>
                    <ClusterIcon className="pf-u-mr-xs" />{' '}
                    <span style={{ position: 'relative', top: '1px' }}>Cluster</span>
                </span>
            }
            toggleAriaLabel="Select a cluster"
            onToggle={toggleIsClusterOpen}
            onSelect={onClusterSelect}
            isOpen={isClusterOpen}
            selections={selectedClusterName}
        >
            {clusterSelectOptions}
        </Select>
    );
}

export default ClusterSelector;
