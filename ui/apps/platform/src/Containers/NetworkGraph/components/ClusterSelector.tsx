import React from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import { Cluster } from 'types/cluster.proto';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { ClusterIcon } from '../common/NetworkGraphIcons';

type ClusterSelectorProps = {
    clusters: Cluster[];
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
            setSearchFilter(modifiedSearchObject);
        }
    };

    const clusterSelectOptions: JSX.Element[] = clusters.map((cluster) => (
        <SelectOption key={cluster.id} value={cluster.name}>
            <span>
                <ClusterIcon /> {cluster.name}
            </span>
        </SelectOption>
    ));

    return (
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
    );
}

export default ClusterSelector;
