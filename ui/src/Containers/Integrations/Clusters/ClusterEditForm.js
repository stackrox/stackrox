import React from 'react';
import PropTypes from 'prop-types';
import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { clusterFormId, clusterTypes } from 'reducers/clusters';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import FormField from 'Components/FormField';
import ReduxTextField from 'Components/forms/ReduxTextField';
import CollapsibleCard from 'Components/CollapsibleCard';
import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';
import InlineToggleField from './InlineToggleField';

const runtimeOptions = [
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

const CommonFields = ({ disabled }) => (
    <React.Fragment>
        <FormField label="Name" required>
            <ReduxTextField disabled={disabled} name="name" placeholder="Cluster name" />
        </FormField>
        <FormField label="Main Image Repository" required>
            <ReduxTextField name="mainImage" placeholder="stackrox.io/main" />
        </FormField>
        <FormField label="Central API Endpoint (include port)" required>
            <ReduxTextField name="centralApiEndpoint" placeholder="central.stackrox:443" />
        </FormField>
        <FormField label="Monitoring Endpoint (include port; empty means no monitoring)">
            <ReduxTextField name="monitoringEndpoint" placeholder="monitoring.stackrox:443" />
        </FormField>
        <FormField label="Collection Method">
            <ReduxSelectField
                key="collectionMethod"
                name="collectionMethod"
                options={runtimeOptions}
            />
        </FormField>
        <FormField label="Collector Image Repository (uses Main image repository by default)">
            <ReduxTextField name="collectorImage" placeholder="collector.stackrox.io/collector" />
        </FormField>
    </React.Fragment>
);

CommonFields.propTypes = {
    disabled: PropTypes.bool.isRequired
};

const DynamicConfig = () => (
    <div className="mt-3">
        <PanelCard title="Dynamic Configuration (syncs with Sensor)">
            <h3>Admission Controller</h3>
            <InlineToggleField
                label="Enable Admission Controller"
                name="dynamicConfig.admissionControllerConfig.enabled"
                borderClass="border-b-2"
            />
            <div className="flex py-1 border-b-2 border-base-300">
                <div className="flex w-full capitalize items-center">Timeout (seconds)</div>
                <ReduxNumericInputField
                    name="dynamicConfig.admissionControllerConfig.timeoutSeconds"
                    min={1}
                    placeholder="3"
                    className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                />
            </div>
            <InlineToggleField
                label="Contact Image Scanners"
                name="dynamicConfig.admissionControllerConfig.scanInline"
                borderClass="border-b-2"
            />
            <InlineToggleField
                label="Disable Use of Bypass Annotation"
                name="dynamicConfig.admissionControllerConfig.disableBypass"
                borderClass="border-b-2"
            />
        </PanelCard>
    </div>
);

const PanelCard = ({ children, open, title }) => (
    <CollapsibleCard
        open={open}
        title={title}
        titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
    >
        <div className="p-3">{children}</div>
    </CollapsibleCard>
);

PanelCard.propTypes = {
    open: PropTypes.bool,
    title: PropTypes.shape({}).isRequired,
    children: PropTypes.node.isRequired
};

PanelCard.defaultProps = {
    open: true
};

const K8sFields = ({ cluster }) => (
    <React.Fragment>
        <PanelCard open={!cluster} title="Static Configuration (requires deployment)">
            <CommonFields disabled={!!cluster} />
            <InlineToggleField
                label="Create Admission Controller Webhook"
                name="admissionController"
            />
        </PanelCard>
        <DynamicConfig />
    </React.Fragment>
);

K8sFields.propTypes = {
    cluster: PropTypes.shape({})
};

K8sFields.defaultProps = {
    cluster: null
};

const OpenShiftFields = ({ cluster }) => (
    <React.Fragment>
        <CommonFields disabled={!!cluster} />
    </React.Fragment>
);

OpenShiftFields.propTypes = {
    cluster: PropTypes.shape({})
};

OpenShiftFields.defaultProps = {
    cluster: null
};

const clusterFields = {
    OPENSHIFT_CLUSTER: OpenShiftFields,
    KUBERNETES_CLUSTER: K8sFields
};

const ClusterEditForm = ({ cluster, clusterType }) => {
    const ClusterFields = clusterFields[clusterType];
    if (!ClusterFields) throw new Error(`Unknown cluster type "${clusterType}"`);
    return (
        <form className="p-4 w-full mb-8" data-test-id="cluster-form">
            <ClusterFields cluster={cluster} />
        </form>
    );
};

ClusterEditForm.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired,
    cluster: PropTypes.shape({}).isRequired
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

const FormWrapper = ({ cluster, clusterType, initialValues, metadata }) => {
    const { releaseBuild } = metadata;
    const combinedInitialValues = {
        ...initialValuesFactories[clusterType],
        mainImage: releaseBuild ? 'stackrox.io/main' : 'stackrox/main',
        collectorImage: releaseBuild ? 'collector.stackrox.io/collector' : 'stackrox/collector',
        type: clusterType,
        ...initialValues, // passed initial values can override anything
        ...cluster
    };

    return (
        <ConnectedForm
            enableReinitialize
            cluster={cluster}
            clusterType={clusterType}
            initialValues={combinedInitialValues}
        />
    );
};
FormWrapper.propTypes = {
    clusterType: PropTypes.oneOf(clusterTypes).isRequired,
    cluster: PropTypes.shape({
        type: PropTypes.oneOf(clusterTypes).isRequired
    }),
    metadata: PropTypes.shape({ version: PropTypes.string, releaseBuild: PropTypes.bool })
        .isRequired,
    initialValues: PropTypes.shape({})
};
FormWrapper.defaultProps = {
    cluster: null,
    initialValues: {}
};

const mapStateToProps = createStructuredSelector({
    metadata: selectors.getMetadata
});

export default connect(mapStateToProps)(FormWrapper);
