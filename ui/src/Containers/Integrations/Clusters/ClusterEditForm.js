import React from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';

import { clusterFormId, clusterTypes } from 'reducers/clusters';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';

const CommonFields = () => (
    <React.Fragment>
        <FormField label="Name" required>
            <ReduxTextField name="name" placeholder="Cluster name" />
        </FormField>
        <FormField label="Image name (Prevent location)" required>
            <ReduxTextField
                name="preventImage"
                placeholder="stackrox.io/prevent:[current-version]"
            />
        </FormField>
    </React.Fragment>
);

const K8sFields = () => (
    <React.Fragment>
        <CommonFields />
        <FormField label="Central API Endpoint" required>
            <ReduxTextField name="centralApiEndpoint" placeholder="central.stackrox:443" />
        </FormField>
        <FormField label="Namespace" required>
            <ReduxTextField name="kubernetes.params.namespace" placeholder="stackrox" />
        </FormField>
        <FormField label="Image Pull Secret Name" required>
            <ReduxTextField name="kubernetes.params.imagePullSecret" placeholder="stackrox" />
        </FormField>
    </React.Fragment>
);

const OpenShiftFields = () => (
    <React.Fragment>
        <CommonFields />
        <FormField label="Central API Endpoint" required>
            <ReduxTextField name="centralApiEndpoint" placeholder="central.stackrox:443" />
        </FormField>
        <FormField label="Namespace" required>
            <ReduxTextField name="openshift.params.namespace" placeholder="stackrox" />
        </FormField>
    </React.Fragment>
);

const DockerFields = () => (
    <React.Fragment>
        <CommonFields />
        <FormField label="Central API Endpoint" required>
            <ReduxTextField name="centralApiEndpoint" placeholder="central.prevent_net:443" />
        </FormField>
        <FormField label="Disable Swarm TLS">
            <ReduxCheckboxField name="swarm.disableSwarmTls" />
        </FormField>
    </React.Fragment>
);

const clusterFields = {
    SWARM_CLUSTER: DockerFields,
    OPENSHIFT_CLUSTER: OpenShiftFields,
    KUBERNETES_CLUSTER: K8sFields
};

const ClusterEditForm = ({ clusterType }) => {
    const ClusterFields = clusterFields[clusterType];
    if (!ClusterFields) throw new Error(`Unknown cluster type "${clusterType}"`);
    return (
        <form className="p-4 w-full mb-8" data-test-id="cluster-form">
            <ClusterFields />
        </form>
    );
};
ClusterEditForm.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired
};

export default reduxForm({ form: clusterFormId })(ClusterEditForm);
