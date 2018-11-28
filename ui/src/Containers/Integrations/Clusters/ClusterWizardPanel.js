import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';

import { actions, wizardPages, clusterTypes } from 'reducers/clusters';
import { selectors } from 'reducers';

import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import ClusterEditForm from './ClusterEditForm';
import ClusterDeploymentPage from './ClusterDeploymentPage';

class ClusterWizardPanel extends Component {
    static propTypes = {
        clusterType: PropTypes.oneOf(clusterTypes).isRequired,
        cluster: PropTypes.shape({
            id: PropTypes.string,
            name: PropTypes.string,
            lastContact: PropTypes.string
        }),
        currentPage: PropTypes.oneOf(Object.values(wizardPages)).isRequired,
        onFinish: PropTypes.func.isRequired,
        onNext: PropTypes.func.isRequired,
        onDownload: PropTypes.func.isRequired,
        metadata: PropTypes.shape({ version: PropTypes.string })
    };

    static defaultProps = {
        cluster: null,
        metadata: {
            version: 'latest'
        }
    };

    componentWillUnmount() {
        this.props.onFinish();
    }

    renderPanelButtons() {
        const { currentPage, onNext, onFinish } = this.props;
        switch (currentPage) {
            case wizardPages.FORM:
                return (
                    <PanelButton
                        icon={<Icon.ArrowRight className="h-4 w-4" />}
                        text="Next"
                        className="btn btn-primary"
                        onClick={onNext}
                    />
                );
            case wizardPages.DEPLOYMENT:
                return (
                    <PanelButton
                        icon={<Icon.Check className="h-4 w-4" />}
                        text="Finish"
                        className="btn btn-success"
                        onClick={onFinish}
                    />
                );
            default:
                throw new Error(`Unknown cluster wizard page ${currentPage}`);
        }
    }

    renderPage() {
        const { currentPage, clusterType, cluster, metadata, onDownload } = this.props;
        switch (currentPage) {
            case wizardPages.FORM:
                return (
                    <ClusterEditForm
                        clusterType={clusterType}
                        initialValues={cluster}
                        metadata={metadata}
                    />
                );
            case wizardPages.DEPLOYMENT:
                return (
                    <ClusterDeploymentPage
                        onFileDownload={onDownload}
                        clusterCheckedIn={!!(cluster && cluster.lastContact)}
                    />
                );
            default:
                throw new Error(`Unknown cluster wizard page ${currentPage}`);
        }
    }

    render() {
        const { cluster, onFinish } = this.props;
        const clusterName = cluster && cluster.name;
        return (
            <div className="w-full">
                <Panel
                    header={clusterName || 'New Cluster'}
                    buttons={this.renderPanelButtons()}
                    className="h-full w-full"
                    onClose={onFinish}
                >
                    {this.renderPage()}
                </Panel>
            </div>
        );
    }
}

const getCluster = createSelector(
    [selectors.getClusters, selectors.getWizardClusterId],
    (clusters, id) => clusters.find(cluster => cluster.id === id)
);

const mapStateToProps = createStructuredSelector({
    cluster: getCluster,
    currentPage: selectors.getWizardCurrentPage,
    metadata: selectors.getMetadata
});

const mapDispatchToProps = {
    onFinish: actions.finishWizard,
    onNext: actions.nextWizardPage,
    onPrev: actions.prevWizardPage,
    onDownload: actions.downloadClusterYaml
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ClusterWizardPanel);
