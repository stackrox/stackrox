import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { ToastContainer, toast } from 'react-toastify';
import * as Icon from 'react-feather';
import Raven from 'raven-js';

import { actions, wizardPages, clusterTypes } from 'reducers/clusters';
import { selectors } from 'reducers';
import { downloadClusterYaml } from 'services/ClustersService';

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

    onDownload = () => {
        downloadClusterYaml(this.props.cluster.id).catch(error => {
            toast('Error while downloading a file');
            Raven.captureException(error);
        });
    };

    renderPanelButtons() {
        switch (this.props.currentPage) {
            case wizardPages.FORM:
                return (
                    <PanelButton
                        icon={<Icon.ArrowRight className="h-4 w-4" />}
                        text="Next"
                        className="btn btn-primary"
                        onClick={this.props.onNext}
                    />
                );
            case wizardPages.DEPLOYMENT:
                return (
                    <PanelButton
                        icon={<Icon.Check className="h-4 w-4" />}
                        text="Finish"
                        className="btn btn-success"
                        onClick={this.props.onFinish}
                    />
                );
            default:
                throw new Error(`Unknown cluster wizard page ${this.props.currentPage}`);
        }
    }

    renderPage() {
        switch (this.props.currentPage) {
            case wizardPages.FORM:
                return (
                    <ClusterEditForm
                        clusterType={this.props.clusterType}
                        initialValues={this.props.cluster}
                        metadata={this.props.metadata}
                    />
                );
            case wizardPages.DEPLOYMENT:
                return (
                    <ClusterDeploymentPage
                        onFileDownload={this.onDownload}
                        clusterCheckedIn={!!this.props.cluster.lastContact}
                    />
                );
            default:
                throw new Error(`Unknown cluster wizard page ${this.props.currentPage}`);
        }
    }

    render() {
        const clusterName = this.props.cluster && this.props.cluster.name;
        return (
            <div className="w-full">
                <ToastContainer
                    toastClassName="font-sans text-base-600 text-base-100 font-600 bg-black"
                    hideProgressBar
                    autoClose={3000}
                />
                <Panel
                    header={clusterName || 'New Cluster'}
                    buttons={this.renderPanelButtons()}
                    className="h-full w-full"
                    onClose={this.props.onFinish}
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
    onPrev: actions.prevWizardPage
};

export default connect(mapStateToProps, mapDispatchToProps)(ClusterWizardPanel);
