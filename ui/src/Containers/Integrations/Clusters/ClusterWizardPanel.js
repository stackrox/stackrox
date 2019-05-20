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
            status: PropTypes.shape({
                lastContact: PropTypes.string
            })
        }),
        currentPage: PropTypes.oneOf(Object.values(wizardPages)).isRequired,
        onFinish: PropTypes.func.isRequired,
        onNext: PropTypes.func.isRequired,
        onDownload: PropTypes.func.isRequired
    };

    static defaultProps = {
        cluster: null
    };

    constructor(props) {
        super(props);
        this.state = {
            editing: false
        };
    }

    componentDidMount() {
        this.setState({ editing: !!this.props.cluster });
    }

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
                        className="btn btn-base"
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
        const { currentPage, clusterType, cluster, onDownload } = this.props;
        const { editing } = this.state;
        switch (currentPage) {
            case wizardPages.FORM:
                if (cluster) {
                    return (
                        <ClusterEditForm
                            clusterType={clusterType}
                            cluster={cluster}
                            initialValues={cluster}
                        />
                    );
                }
                return <ClusterEditForm clusterType={clusterType} initialValues={cluster} />;
            case wizardPages.DEPLOYMENT:
                return (
                    <ClusterDeploymentPage
                        editing={editing}
                        onFileDownload={onDownload}
                        clusterCheckedIn={
                            !!(cluster && cluster.status && cluster.status.lastContact)
                        }
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
                    headerComponents={this.renderPanelButtons()}
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
    currentPage: selectors.getWizardCurrentPage
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
