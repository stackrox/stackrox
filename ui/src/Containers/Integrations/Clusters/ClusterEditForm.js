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

const StackRoxImageFormField = () => (
    <FormField label="Main Image Repository" required>
        <ReduxTextField name="mainImage" placeholder="stackrox.io/main" />
    </FormField>
);

const RuntimeSupportFormField = () => (
    <FormField label="Runtime Support">
        <ReduxCheckboxField name="runtimeSupport" />
    </FormField>
);

const MonitoringEndpointFormField = () => (
    <FormField label="Monitoring Endpoint (include port; empty means no monitoring)">
        <ReduxTextField name="monitoringEndpoint" placeholder="monitoring.stackrox:443" />
    </FormField>
);

const K8sFields = () => (
    <React.Fragment>
        <CommonFields />
        <StackRoxImageFormField />
        <CentralAPIFormField placeholder="central.stackrox:443" />
        <MonitoringEndpointFormField />
        <RuntimeSupportFormField />
        <FormField label="Enable Admission Controller">
            <ReduxCheckboxField name="admissionController" />
        </FormField>
    </React.Fragment>
);

const OpenShiftFields = () => (
    <React.Fragment>
        <CommonFields />
        <StackRoxImageFormField />
        <CentralAPIFormField placeholder="central.stackrox:443" />
        <MonitoringEndpointFormField />
        <RuntimeSupportFormField />
    </React.Fragment>
);

const clusterFields = {
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

const ConnectedForm = reduxForm({ form: clusterFormId })(ClusterEditForm);

const initialValuesFactories = {
    OPENSHIFT_CLUSTER: {
        mainImage: `stackrox.io/main`,
        centralApiEndpoint: 'central.stackrox:443',
        monitoringEndpoint: 'monitoring.stackrox:443',
        runtimeSupport: true
    },
    KUBERNETES_CLUSTER: {
        mainImage: `stackrox.io/main`,
        centralApiEndpoint: 'central.stackrox:443',
        monitoringEndpoint: 'monitoring.stackrox:443',
        runtimeSupport: true,
        admissionController: false
    }
};

const FormWrapper = ({ clusterType, initialValues }) => {
    const combinedInitialValues = {
        ...initialValuesFactories[clusterType],
        type: clusterType,
        ...initialValues // passed initial values can override anything
    };

    return <ConnectedForm clusterType={clusterType} initialValues={combinedInitialValues} />;
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
