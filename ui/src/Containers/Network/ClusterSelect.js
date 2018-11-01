import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as networkActions, networkGraphClusters } from 'reducers/network';
import { actions as clusterActions } from 'reducers/clusters';

import Select from 'Components/ReactSelect';

class ClusterSelect extends Component {
    static propTypes = {
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedClusterId: PropTypes.string,
        selectClusterId: PropTypes.func.isRequired,
        fetchClusters: PropTypes.func.isRequired,
        closeSidePanel: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedClusterId: ''
    };

    changeCluster = clusterId => {
        this.props.selectClusterId(clusterId);
        this.props.closeSidePanel();
    };

    render() {
        if (!this.props.clusters.length) return null;
        // network policies are only applicable on k8s-based clusters
        const options = this.props.clusters
            .filter(cluster => networkGraphClusters[cluster.type])
            .map(cluster => ({
                value: cluster.id,
                label: cluster.name
            }));
        const clustersProps = {
            className: 'min-w-64 ml-2',
            options,
            value: this.props.selectedClusterId,
            placeholder: 'Select a cluster',
            onChange: this.changeCluster,
            autoFocus: true
        };
        return <Select {...clustersProps} />;
    }
}

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId
});

const mapDispatchToProps = {
    fetchClusters: clusterActions.fetchClusters.request,
    selectClusterId: networkActions.selectNetworkClusterId
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ClusterSelect);
