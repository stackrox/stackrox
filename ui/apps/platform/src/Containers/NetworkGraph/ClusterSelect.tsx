import React, { ReactElement } from 'react';
import { Select, SelectOption } from '@patternfly/react-core';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Cluster } from 'types/cluster.proto';

type ClusterSelectProps = {
    id?: string;
    setSelectedClusterId: (clusterId: string) => void;
    clusters: Cluster[];
    selectedClusterId?: string;
    isDisabled?: boolean;
};

const ClusterSelect = ({
    id,
    setSelectedClusterId,
    clusters,
    selectedClusterId = '',
    isDisabled = false,
}: ClusterSelectProps): ReactElement => {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    function changeCluster(_e, clusterId) {
        setSelectedClusterId(clusterId);
        closeSelect();
    }

    return (
        <Select
            id={id}
            isOpen={isOpen}
            onToggle={onToggle}
            isDisabled={isDisabled || !clusters.length}
            selections={selectedClusterId}
            placeholderText="Select a cluster"
            onSelect={changeCluster}
        >
            {clusters.map(({ id: clusterId, name }) => (
                <SelectOption key={clusterId} value={clusterId}>
                    {name}
                </SelectOption>
            ))}
        </Select>
    );
};

export default ClusterSelect;
