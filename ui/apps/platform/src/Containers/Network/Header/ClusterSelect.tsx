import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Select, SelectOption } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as graphActions, networkGraphClusters } from 'reducers/network/graph';
import { actions as pageActions } from 'reducers/network/page';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Cluster } from 'types/cluster.proto';

type ClusterSelectProps = {
    id?: string;
    selectClusterId: (clusterId: string) => void;
    closeSidePanel: () => void;
    clusters: Cluster[];
    selectedClusterId?: string;
    isDisabled?: boolean;
};

const ClusterSelect = ({
    id,
    selectClusterId,
    closeSidePanel,
    clusters,
    selectedClusterId = '',
    isDisabled = false,
}: ClusterSelectProps): ReactElement => {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    function changeCluster(_e, clusterId) {
        selectClusterId(clusterId);
        closeSelect();
        closeSidePanel();
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
            {clusters
                .filter((cluster) => networkGraphClusters[cluster.type])
                .map(({ id: clusterId, name }) => (
                    <SelectOption key={clusterId} value={clusterId}>
                        {name}
                    </SelectOption>
                ))}
        </Select>
    );
};

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId,
});

const mapDispatchToProps = {
    selectClusterId: graphActions.selectNetworkClusterId,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterSelect);
