import React from 'react';
import PropTypes from 'prop-types';

import dateFns from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { clusterTypes } from 'reducers/clusters';
import LabeledValue from 'Components/LabeledValue';

const clusterTypeLabels = {
    OPENSHIFT_CLUSTER: 'OpenShift',
    KUBERNETES_CLUSTER: 'Kubernetes'
};

const enabledOrDisabled = value => (value ? 'Enabled' : 'Disabled');

export const formatRuntimeSupport = cluster => enabledOrDisabled(cluster.runtimeSupport);
export const formatAdmissionController = cluster => enabledOrDisabled(cluster.admissionController);

export const checkInLabel = 'Last Check-In';
export const formatLastCheckIn = cluster => {
    if (cluster.status && cluster.status.lastContact) {
        return dateFns.format(cluster.status.lastContact, dateTimeFormat);
    }
    return 'N/A';
};

export const sensorVersionLabel = 'Current Sensor Version';
export const formatSensorVersion = cluster =>
    (cluster.status && cluster.status.sensorVersion) || 'Not Running';

const CommonDetails = ({ cluster }) => (
    <React.Fragment>
        <LabeledValue label="Name" value={cluster.name} />
        <LabeledValue label="Cluster Type" value={clusterTypeLabels[cluster.type]} />
        <LabeledValue label="Original Configured Image" value={cluster.mainImage} />
        <LabeledValue label="Central API Endpoint" value={cluster.centralApiEndpoint} />
        <LabeledValue label="Runtime Support" value={formatRuntimeSupport(cluster)} />
        <LabeledValue label={checkInLabel} value={formatLastCheckIn(cluster)} />
        <LabeledValue label={sensorVersionLabel} value={formatSensorVersion(cluster)} />
    </React.Fragment>
);
CommonDetails.propTypes = {
    cluster: PropTypes.shape({
        name: PropTypes.string.isRequired,
        type: PropTypes.string.isRequired,
        mainImage: PropTypes.string.isRequired,
        centralApiEndpoint: PropTypes.string.isRequired,
        runtimeSupport: PropTypes.bool.isRequired
    }).isRequired
};

const K8sDetails = ({ cluster }) => (
    <React.Fragment>
        <CommonDetails cluster={cluster} />
        <LabeledValue label="Monitoring Endpoint" value={cluster.monitoringEndpoint || 'N/A'} />
        <LabeledValue label="Admission Controller" value={formatAdmissionController(cluster)} />
    </React.Fragment>
);
K8sDetails.propTypes = {
    cluster: PropTypes.shape({}).isRequired
};

const OpenShiftDetails = ({ cluster }) => (
    <React.Fragment>
        <CommonDetails cluster={cluster} />
    </React.Fragment>
);
OpenShiftDetails.propTypes = {
    cluster: PropTypes.shape({}).isRequired
};

const detailsComponents = {
    OPENSHIFT_CLUSTER: OpenShiftDetails,
    KUBERNETES_CLUSTER: K8sDetails
};

const ClusterDetails = ({ cluster }) => {
    const DetailsComponent = detailsComponents[cluster.type];
    if (!DetailsComponent) throw new Error(`Unknown cluster type "${cluster.type}"`);
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
