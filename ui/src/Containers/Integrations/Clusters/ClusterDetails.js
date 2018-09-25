import React from 'react';
import PropTypes from 'prop-types';
import get from 'lodash/get';

import { clusterTypes } from 'reducers/clusters';
import LabeledValue from 'Components/LabeledValue';

const clusterTypeLabels = {
    SWARM_CLUSTER: 'Swarm',
    OPENSHIFT_CLUSTER: 'OpenShift',
    KUBERNETES_CLUSTER: 'Kubernetes'
};

const CommonDetails = ({ cluster }) => (
    <React.Fragment>
        <LabeledValue label="Name" value={cluster.name} />
        <LabeledValue label="Cluster Type" value={clusterTypeLabels[cluster.type]} />
        <LabeledValue label="Prevent Image" value={cluster.preventImage} />
        <LabeledValue label="Central API Endpoint" value={cluster.centralApiEndpoint} />
        <LabeledValue
            label="Runtime Support"
            value={cluster.runtimeSupport ? 'Enabled' : 'Disabled'}
        />
    </React.Fragment>
);
CommonDetails.propTypes = {
    cluster: PropTypes.shape({
        name: PropTypes.string.isRequired,
        type: PropTypes.string.isRequired,
        preventImage: PropTypes.string.isRequired,
        centralApiEndpoint: PropTypes.string.isRequired,
        runtimeSupport: PropTypes.bool.isRequired
    }).isRequired
};

const K8sDetails = ({ cluster }) => (
    <React.Fragment>
        <CommonDetails cluster={cluster} />
        <LabeledValue
            label="Namespace"
            value={get(cluster, 'kubernetes.params.namespace', 'N/A')}
        />
        <LabeledValue
            label="Image Pull Secret Name"
            value={get(cluster, 'kubernetes.imagePullSecret', 'N/A')}
        />
    </React.Fragment>
);
K8sDetails.propTypes = {
    cluster: PropTypes.shape({
        kubernetes: PropTypes.shape({
            params: PropTypes.shape({
                namespace: PropTypes.string,
                imagePullSecret: PropTypes.string
            })
        })
    }).isRequired
};

const OpenShiftDetails = ({ cluster }) => (
    <React.Fragment>
        <CommonDetails cluster={cluster} />
        <LabeledValue label="Namespace" value={get(cluster, 'openshift.params.namespace', 'N/A')} />
    </React.Fragment>
);
OpenShiftDetails.propTypes = {
    cluster: PropTypes.shape({
        openshift: PropTypes.shape({
            params: PropTypes.shape({ namespace: PropTypes.string })
        })
    }).isRequired
};

const DockerDetails = ({ cluster }) => (
    <React.Fragment>
        <CommonDetails cluster={cluster} />
        <LabeledValue
            label="Swarm TLS Disabled"
            value={get(cluster, 'swarm.disableSwarmTls') ? 'Yes' : 'No'}
        />
    </React.Fragment>
);
DockerDetails.propTypes = {
    cluster: PropTypes.shape({
        swarm: PropTypes.shape({ disableSwarmTls: PropTypes.bool })
    }).isRequired
};

const detailsComponents = {
    SWARM_CLUSTER: DockerDetails,
    OPENSHIFT_CLUSTER: OpenShiftDetails,
    KUBERNETES_CLUSTER: K8sDetails
};

const ClusterDetails = ({ cluster }) => {
    const DetailsComponent = detailsComponents[cluster.type];
    if (!DetailsComponent) throw new Error(`Unknown cluster type "${this.props.clusterType}"`);
    return (
        <div className="p-4">
            <DetailsComponent cluster={cluster} />
        </div>
    );
};
ClusterDetails.propTypes = {
    cluster: PropTypes.shape({
        type: PropTypes.oneOf(clusterTypes).isRequired
    }).isRequired
};

export default ClusterDetails;
