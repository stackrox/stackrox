import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { actions as clusterActions } from 'reducers/clusters';
import { selectors } from 'reducers';
import { submit, formValueSelector } from 'redux-form';
import { createCluster } from 'services/ClustersService';

import Panel from 'Components/Panel';
import PanelSlider from 'Components/PanelSlider';
import SimpleForm from 'Components/SimpleForm';
import KeyValuePairs from 'Components/KeyValuePairs';
import ClustersDownloadPage from 'Containers/Integrations/ClustersDownloadPage';
import ClustersSuccessPage from 'Containers/Integrations/ClustersSuccessPage';
import clusterCreationFormDescriptor from 'Containers/Integrations/clusterCreationFormDescriptor';
import { ToastContainer, toast } from 'react-toastify';

const clusterDetailsMap = {
    name: {
        label: 'Name'
    },
    type: {
        label: 'Cluster Type'
    },
    preventImage: {
        label: 'Image name (Prevent location)'
    },
    centralApiEndpoint: {
        label: 'Central API Endpoint'
    },
    namespace: {
        label: 'Namespace'
    },
    imagePullSecret: {
        label: 'Image Pull Secret Name'
    },
    disableSwarmTls: {
        label: 'Swarm TLS Disabled'
    }
};

const formDataKeys = clusterCreationFormDescriptor.map(obj => obj.value);

class ClusterCreationPanel extends Component {
    static propTypes = {
        editingCluster: PropTypes.shape(),
        editCluster: PropTypes.func.isRequired,
        isClusterSuccessfullyConfigured: PropTypes.bool,
        formData: PropTypes.shape().isRequired,
        setCreatedClusterId: PropTypes.func.isRequired,
        createdClusterId: PropTypes.string
    };

    static defaultProps = {
        editingCluster: null,
        createdClusterId: null,
        isClusterSuccessfullyConfigured: null
    };

    onFinish = () => {
        this.props.editCluster(null);
    };

    onCancelEdit = () => {
        this.props.editCluster(null);
    };

    onDownload = () => {
        this.downloadPage.downloadYamlFile(this.props.createdClusterId);
    };

    onNext = index => {
        const promise = new Promise((resolve, reject) => {
            if (index === 0) {
                const createClusterPromise = createCluster(this.props.formData);
                createClusterPromise
                    .then(result => {
                        this.props.setCreatedClusterId(result.data.cluster.id);
                        resolve();
                    })
                    .catch(error => {
                        toast(error.response.data.error);
                        reject();
                    });
            } else {
                resolve();
            }
        });
        return promise;
    };

    renderKeyValuePairs = () => (
        <div className="p-4">
            <KeyValuePairs data={this.props.editingCluster} keyValueMap={clusterDetailsMap} />
        </div>
    );

    renderFormPanel = () => {
        let fields = clusterCreationFormDescriptor;
        // if viewing an existing cluster, disable the fields
        if (this.props.editingCluster.id) {
            fields = fields.map(field => {
                const result = Object.assign({}, field);
                result.disabled = true;
                return result;
            });
        }
        return (
            <SimpleForm
                fields={fields}
                onSubmit={this.submit}
                initialValues={this.props.editingCluster}
            />
        );
    };

    renderDownloadPagePanel = () => {
        if (this.props.editingCluster.id) return null;
        return (
            <div>
                <ClustersDownloadPage
                    cluster={this.props.editingCluster}
                    onClick={this.onDownload}
                    ref={downloadPage => {
                        this.downloadPage = downloadPage;
                    }}
                />
                <ClustersSuccessPage success={this.props.isClusterSuccessfullyConfigured} />
            </div>
        );
    };

    render() {
        if (this.props.editingCluster.id) {
            return (
                <Panel
                    className="flex flex-col w-full"
                    header={this.props.editingCluster.id}
                    onClose={this.onCancelEdit}
                >
                    {this.renderKeyValuePairs()}
                </Panel>
            );
        }
        return (
            <div className="w-full">
                <ToastContainer
                    toastClassName="font-sans text-base-600 text-white font-600 bg-black"
                    hideProgressBar
                    autoClose={3000}
                />
                <PanelSlider
                    className="h-full w-full"
                    header="New Cluster"
                    onFinish={this.onFinish}
                    onClose={this.onCancelEdit}
                    disablePrevious
                    onNext={this.onNext}
                >
                    <div>{this.renderFormPanel()}</div>
                    <div>{this.renderDownloadPagePanel()}</div>
                </PanelSlider>
            </div>
        );
    }
}

const getEditingCluster = createSelector(
    [selectors.getClusters, selectors.getEditingCluster],
    (clusters, editingCluster) => {
        if (!editingCluster) {
            return null;
        }
        let result = {};
        if (!editingCluster.id) {
            return result;
        }
        result = clusters.find(obj => obj.id === editingCluster.id);
        return result;
    }
);

const getSuccessfullyConfiguredCluster = createSelector(
    [selectors.getClusters, selectors.getCreatedClusterId],
    (clusters, clusterId) => {
        const unconfiguredCluster = clusters.find(
            obj => obj.id === clusterId && obj.lastContact !== null
        );
        return unconfiguredCluster;
    }
);

const getFormData = createSelector(
    [state => formValueSelector('simpleform')(state, ...formDataKeys)],
    formData => formData
);

const mapStateToProps = createStructuredSelector({
    editingCluster: getEditingCluster,
    isClusterSuccessfullyConfigured: getSuccessfullyConfiguredCluster,
    formData: getFormData,
    createdClusterId: selectors.getCreatedClusterId
});

const mapDispatchToProps = dispatch => ({
    saveCluster: cluster => dispatch(clusterActions.saveCluster(cluster)),
    submitForm: () => dispatch(submit('simpleform')),
    editCluster: clusterId => dispatch(clusterActions.editCluster(clusterId)),
    setCreatedClusterId: clusterId => dispatch(clusterActions.setCreatedClusterId(clusterId))
});

export default connect(mapStateToProps, mapDispatchToProps)(ClusterCreationPanel);
