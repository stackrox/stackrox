import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Select, SelectOption } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';
import { actions as pageActions } from 'reducers/network/page';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';

type ClusterSelectProps = {
    id?: string;
    selectClusterId: (clusterId: string) => void;
    closeSidePanel: () => void;
    selectedClusterId?: string;
    isDisabled?: boolean;
};

const ClusterSelect = ({
    id,
    selectClusterId,
    closeSidePanel,
    selectedClusterId = '',
    isDisabled = false,
}: ClusterSelectProps): ReactElement => {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    function changeCluster(_e, clusterId) {
        selectClusterId(clusterId);
        closeSelect();
        closeSidePanel();
    }
    const { clusters } = useFetchClustersForPermissions(['NetworkGraph', 'Deployment']);

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

const mapStateToProps = createStructuredSelector({
    selectedClusterId: selectors.getSelectedNetworkClusterId,
});

const mapDispatchToProps = {
    selectClusterId: graphActions.selectNetworkClusterId,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterSelect);
