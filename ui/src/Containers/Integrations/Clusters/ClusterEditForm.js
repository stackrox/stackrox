import React from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';

import { clusterFormId, clusterTypes } from 'reducers/clusters';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxCheckboxField from 'Components/forms/ReduxCheckboxField';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import ReduxSelectField from 'Components/forms/ReduxSelectField';

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

const StackRoxCollectorImageFormField = () => (
    <FormField label="Collector Image Repository (uses Main image repository by default)">
        <ReduxTextField name="collectorImage" placeholder="collector.stackrox.io/collector" />
    </FormField>
);

const RuntimeSupportFormField = () => {
    const options = [
        {
            label: 'No Runtime Support',
            value: 'NO_COLLECTION'
        },
        {
            label: 'Kernel Module Support',
            value: 'KERNEL_MODULE'
        },
        {
            label: '[BETA] eBPF Support',
            value: 'EBPF'
        }
    ];
    return (
        <FormField label="Collection Method">
            <ReduxSelectField key="collectionMethod" name="collectionMethod" options={options} />
        </FormField>
    );
};

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
        <StackRoxCollectorImageFormField />
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
        <StackRoxCollectorImageFormField />
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
        centralApiEndpoint: 'central.stackrox:443',
        monitoringEndpoint: 'monitoring.stackrox:443',
        collectionMethod: 'KERNEL_MODULE',
        collectorImage: `collector.stackrox.io/collector`
    },
    KUBERNETES_CLUSTER: {
        centralApiEndpoint: 'central.stackrox:443',
        monitoringEndpoint: 'monitoring.stackrox:443',
        collectionMethod: 'KERNEL_MODULE',
        collectorImage: `collector.stackrox.io/collector`,
        admissionController: false
    }
};

const FormWrapper = ({ clusterType, initialValues, metadata }) => {
    const { releaseBuild } = metadata;
    const combinedInitialValues = {
        ...initialValuesFactories[clusterType],
        mainImage: releaseBuild ? 'stackrox.io/main' : 'stackrox/main',
        collectorImage: releaseBuild ? 'collector.stackrox.io/collector' : 'stackrox/collector',
        type: clusterType,
        ...initialValues // passed initial values can override anything
    };

    return <ConnectedForm clusterType={clusterType} initialValues={combinedInitialValues} />;
};
FormWrapper.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired,
    metadata: PropTypes.shape({ version: PropTypes.string, releaseBuild: PropTypes.bool })
        .isRequired,
    initialValues: PropTypes.shape({})
};
FormWrapper.defaultProps = {
    initialValues: {}
};

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata
});

export default connect(mapStateToProps)(FormWrapper);
