import React, { ReactElement, useCallback, useEffect } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Select, SelectOption } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as graphActions, networkGraphClusters } from 'reducers/network/graph';
import { actions as pageActions } from 'reducers/network/page';

import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { Cluster } from 'types/cluster.proto';
import useURLParameter from 'hooks/useURLParameter';

type ClusterSelectProps = {
    id?: string;
    selectClusterId: (clusterId: string) => void;
    closeSidePanel: () => void;
    clusters: Cluster[];
    selectedClusterId?: string;
    isDisabled?: boolean;
};

// TODO Are there use cases where we want the possibility of multiple selected clusters,
// and should that be rolled into this hook?
// TODO extract
function useURLCluster(defaultClusterId: string) {
    const [cluster, setClusterInternal] = useURLParameter('cluster', defaultClusterId || undefined);
    const setCluster = useCallback(
        (clusterId?: string) => {
            setClusterInternal(clusterId);
        },
        [setClusterInternal]
    );

    return {
        cluster: typeof cluster === 'string' ? cluster : undefined,
        setCluster,
    };
}

const ClusterSelect = ({
    id,
    selectClusterId,
    closeSidePanel,
    clusters,
    selectedClusterId = '',
    isDisabled = false,
}: ClusterSelectProps): ReactElement => {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    const { cluster, setCluster } = useURLCluster(selectedClusterId);

    useEffect(() => {
        selectClusterId(cluster || '');
    }, [cluster, selectClusterId]);

    function changeCluster(_e, clusterId) {
        selectClusterId(clusterId);
        closeSelect();
        closeSidePanel();
        setCluster(clusterId);
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
                .filter((c) => networkGraphClusters[c.type])
                .map(({ id: clusterId, name }) => (
                    <SelectOption
                        isSelected={clusterId === cluster}
                        key={clusterId}
                        value={clusterId}
                    >
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
