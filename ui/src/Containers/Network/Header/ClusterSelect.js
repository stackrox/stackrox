import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as graphActions, networkGraphClusters } from 'reducers/network/graph';
import { actions as clusterActions } from 'reducers/clusters';
import { actions as pageActions } from 'reducers/network/page';

import Select from 'Components/ReactSelect';

const ClusterSelect = ({ selectClusterId, closeSidePanel, clusters, selectedClusterId }) => {
    function changeCluster(clusterId) {
        selectClusterId(clusterId);
        closeSidePanel();
    }

    if (!clusters.length) return null;
    // network policies are only applicable on k8s-based clusters
    const options = clusters
        .filter(cluster => networkGraphClusters[cluster.type])
        .map(cluster => ({
            value: cluster.id,
            label: cluster.name
        }));
    const clustersProps = {
        className: 'min-w-48',
        options,
        value: selectedClusterId,
        placeholder: 'Select a cluster',
        onChange: changeCluster,
        autoFocus: true
    };
    return <Select {...clustersProps} />;
};

ClusterSelect.propTypes = {
    clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
    selectedClusterId: PropTypes.string,
    selectClusterId: PropTypes.func.isRequired,
    fetchClusters: PropTypes.func.isRequired,
    closeSidePanel: PropTypes.func.isRequired
};

ClusterSelect.defaultProps = {
    selectedClusterId: ''
};

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId
});

const mapDispatchToProps = {
    fetchClusters: clusterActions.fetchClusters.request,
    selectClusterId: graphActions.selectNetworkClusterId,
    closeSidePanel: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ClusterSelect);
