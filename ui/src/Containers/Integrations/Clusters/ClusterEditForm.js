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
    </React.Fragment>
);

const CentralAPIFormField = ({ placeholder }) => (
    <FormField label="Central API Endpoint (include port)" required>
        <ReduxTextField name="centralApiEndpoint" placeholder={`${placeholder}`} />
    </FormField>
);

CentralAPIFormField.propTypes = {
    placeholder: PropTypes.string.isRequired
};

const StackRoxImageFormField = ({ version }) => (
    <FormField label="StackRox Image" required>
        <ReduxTextField name="mainImage" placeholder={`stackrox.io/main:${version}`} />
    </FormField>
);

StackRoxImageFormField.propTypes = {
    version: PropTypes.string.isRequired
};

const RuntimeSupportFormField = () => (
    <FormField label="Runtime Support">
        <ReduxCheckboxField name="runtimeSupport" />
    </FormField>
);

const MonitoringEndpointFormField = () => (
    <FormField label="Monitoring Endpoint (empty means no monitoring)">
        <ReduxTextField name="monitoringEndpoint" placeholder="monitoring.stackrox" />
    </FormField>
);

const K8sFields = ({ metadata }) => (
    <React.Fragment>
        <CommonFields />
        <StackRoxImageFormField version={metadata.version} />
        <CentralAPIFormField placeholder="central.stackrox:443" />
        <FormField label="Namespace" required>
            <ReduxTextField name="kubernetes.params.namespace" placeholder="stackrox" />
        </FormField>
        <MonitoringEndpointFormField />
        <RuntimeSupportFormField />
    </React.Fragment>
);

K8sFields.propTypes = {
    metadata: PropTypes.shape({ version: PropTypes.string }).isRequired
};

const OpenShiftFields = ({ metadata }) => (
    <React.Fragment>
        <CommonFields />
        <StackRoxImageFormField version={metadata.version} />
        <CentralAPIFormField placeholder="central.stackrox:443" />
        <FormField label="Namespace" required>
            <ReduxTextField name="openshift.params.namespace" placeholder="stackrox" />
        </FormField>
        <MonitoringEndpointFormField />
        <RuntimeSupportFormField />
    </React.Fragment>
);

OpenShiftFields.propTypes = {
    metadata: PropTypes.shape({ version: PropTypes.string }).isRequired
};

const DockerFields = ({ metadata }) => (
    <React.Fragment>
        <CommonFields />
        <StackRoxImageFormField version={metadata.version} />
        <CentralAPIFormField placeholder="central.prevent_net:443" />
        <FormField label="Disable Swarm TLS">
            <ReduxCheckboxField name="swarm.disableSwarmTls" />
        </FormField>
    </React.Fragment>
);

DockerFields.propTypes = {
    metadata: PropTypes.shape({ version: PropTypes.string }).isRequired
};

const clusterFields = {
    SWARM_CLUSTER: DockerFields,
    OPENSHIFT_CLUSTER: OpenShiftFields,
    KUBERNETES_CLUSTER: K8sFields
};

const ClusterEditForm = ({ clusterType, metadata }) => {
    const ClusterFields = clusterFields[clusterType];
    if (!ClusterFields) throw new Error(`Unknown cluster type "${clusterType}"`);
    return (
        <form className="p-4 w-full mb-8" data-test-id="cluster-form">
            <ClusterFields metadata={metadata} />
        </form>
    );
};
ClusterEditForm.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired,
    metadata: PropTypes.shape({ version: PropTypes.string }).isRequired
};

const ConnectedForm = reduxForm({ form: clusterFormId })(ClusterEditForm);

const initialValuesFactories = {
    SWARM_CLUSTER: metadata => ({
        mainImage: `stackrox.io/main:${metadata.version}`,
        centralApiEndpoint: 'central.prevent_net:443'
    }),
    OPENSHIFT_CLUSTER: metadata => ({
        mainImage: `stackrox.io/main:${metadata.version}`,
        centralApiEndpoint: 'central.stackrox:443',
        openshift: {
            params: {
                namespace: 'stackrox'
            }
        },
        runtimeSupport: true
    }),
    KUBERNETES_CLUSTER: metadata => ({
        mainImage: `stackrox.io/main:${metadata.version}`,
        centralApiEndpoint: 'central.stackrox:443',
        kubernetes: {
            imagePullSecret: 'stackrox',
            params: {
                namespace: 'stackrox'
            }
        },
        monitoringEndpoint: 'monitoring.stackrox',
        runtimeSupport: true
    })
};

const FormWrapper = ({ metadata, clusterType, initialValues }) => {
    const combinedInitialValues = {
        ...initialValuesFactories[clusterType](metadata),
        type: clusterType,
        ...initialValues // passed initial values can override anything
    };

    return (
        <ConnectedForm
            clusterType={clusterType}
            metadata={metadata}
            initialValues={combinedInitialValues}
        />
    );
};
FormWrapper.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired,
    metadata: PropTypes.shape({ version: PropTypes.string }).isRequired,
    initialValues: PropTypes.shape({})
};
FormWrapper.defaultProps = {
    initialValues: {}
};

export default FormWrapper;
