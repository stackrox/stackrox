import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Select, SelectOption } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as graphActions, networkGraphClusters } from 'reducers/network/graph';
import { actions as clusterActions } from 'reducers/clusters';
import { actions as pageActions } from 'reducers/network/page';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';

const ClusterSelect = ({
    selectClusterId,
    closeSidePanel,
    clusters,
    selectedClusterId,
    isDisabled,
}) => {
    const { closeSelect, isOpen, onToggle } = useSelectToggle();
    function changeCluster(_e, clusterId) {
        selectClusterId(clusterId);
        closeSelect();
        closeSidePanel();
    }

    if (!clusters.length) {
        return null;
    }

    return (
        <Select
            isOpen={isOpen}
            onToggle={onToggle}
            isDisabled={isDisabled}
            selections={selectedClusterId}
            placeholderText="Select a cluster"
            onSelect={changeCluster}
        >
            {clusters
                .filter((cluster) => networkGraphClusters[cluster.type])
                .map(({ id, name }) => (
                    <SelectOption key={id} value={id}>
                        {name}
                    </SelectOption>
                ))}
        </Select>
    );
};

ClusterSelect.propTypes = {
    clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedClusterId: PropTypes.string,
    selectClusterId: PropTypes.func.isRequired,
    closeSidePanel: PropTypes.func.isRequired,
    isDisabled: PropTypes.bool,
};

ClusterSelect.defaultProps = {
    selectedClusterId: '',
    isDisabled: false,
};

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId,
});

const mapDispatchToProps = {
    fetchClusters: clusterActions.fetchClusters.request,
    selectClusterId: graphActions.selectNetworkClusterId,
    closeSidePanel: pageActions.closeSidePanel,
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterSelect);
