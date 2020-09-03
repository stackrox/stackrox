import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';
import FormFieldRequired from 'Components/forms/FormFieldRequired';
import Loader from 'Components/Loader';
import Select from 'Components/Select';
import ToggleSwitch from 'Components/ToggleSwitch';
import Message from 'Components/Message';
import FeatureEnabled from 'Containers/FeatureEnabled';
import { knownBackendFlags } from 'utils/featureFlags';

import { clusterTypeOptions, runtimeOptions } from './cluster.helpers';
import ClusterHealth from './Components/ClusterHealth';

const labelClassName = 'block py-2 text-base-600 font-700';
const sublabelClassName = 'font-600 italic';

const inputBaseClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal p-2 rounded text-base-600';
const inputTextClassName = `${inputBaseClassName} w-full`;
const inputNumberClassName = `${inputBaseClassName} text-right w-12`;

// The select element base style includes: pr-8 w-full
const selectElementClassName =
    'bg-base-100 block border-base-300 focus:border-base-500 p-2 text-base-600 z-1';
const selectWrapperClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal rounded text-base-600 w-full';

const divToggleOuterClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal mb-4 px-2 py-2 rounded text-base-600 w-full';

const justifyBetweenClassName = 'flex items-center justify-between';

// factory that returns a handler to normalize our generic Select component's return value
function getSelectComparison(options, key, selectedCluster, handleChange) {
    return function compareSelected(selectedOption) {
        if (
            options.find((value) => value === selectedOption) !== undefined &&
            selectedCluster[key] !== selectedOption.value
        ) {
            const syntheticEvent = {
                target: {
                    name: key,
                    value: selectedOption.value,
                },
            };

            handleChange(syntheticEvent);
        }
    };
}

function ClusterEditForm({ centralEnv, centralVersion, selectedCluster, handleChange, isLoading }) {
    // curry the change handlers for the select inputs
    const onCollectionMethodChange = getSelectComparison(
        runtimeOptions,
        'collectionMethod',
        selectedCluster,
        handleChange
    );
    const onClusterTypeChange = getSelectComparison(
        clusterTypeOptions,
        'type',
        selectedCluster,
        handleChange
    );

    const renderSlimCollectorWarning = () => {
        if (!centralEnv?.successfullyFetched) {
            return (
                <Message
                    message="Failed to check if Central has kernel support packages available"
                    type="warn"
                />
            );
        }

        if (selectedCluster.slimCollector && !centralEnv.kernelSupportAvailable) {
            return (
                <Message
                    message={
                        <span>
                            Central doesn&apos;t have the required Kernel support package. Retrieve
                            it from{' '}
                            <a
                                href="https://install.stackrox.io/collector/support-packages/index.html"
                                className="underline text-primary-900"
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                stackrox.io
                            </a>{' '}
                            and upload it to Central using roxctl.
                        </span>
                    }
                    type="warn"
                />
            );
        }

        return null;
    };

    if (isLoading) {
        return <Loader />;
    }

    return (
        <form className="px-4 w-full mb-8" data-testid="cluster-form">
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {selectedCluster.id && (
                <CollapsibleCard
                    open
                    title="Cluster Health"
                    cardClassName="border border-base-400 mb-2"
                    titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
                >
                    <div className="p-3">
                        <div className="mb-4">
                            <ClusterHealth
                                healthStatus={selectedCluster.healthStatus}
                                status={selectedCluster.status}
                                centralVersion={centralVersion}
                                currentDatetime={new Date()}
                            />
                        </div>
                    </div>
                </CollapsibleCard>
            )}
            <CollapsibleCard
                open
                title="Static Configuration (requires deployment)"
                cardClassName="border border-base-400 mb-2"
                titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
            >
                <div className="p-3">
                    <div className="mb-4">
                        <label htmlFor="name" className={labelClassName}>
                            Cluster Name{' '}
                            <FormFieldRequired empty={selectedCluster.name.length === 0} />
                        </label>
                        <input
                            id="name"
                            name="name"
                            value={selectedCluster.name}
                            onChange={handleChange}
                            disabled={selectedCluster.id}
                            className={inputTextClassName}
                        />
                    </div>
                    <div className="mb-4">
                        <label htmlFor="clusterType" className={labelClassName}>
                            Cluster Type{' '}
                            <FormFieldRequired empty={selectedCluster.type.length === 0} />
                        </label>
                        <Select
                            id="clusterType"
                            options={clusterTypeOptions}
                            placeholder="Select a cluster type"
                            onChange={onClusterTypeChange}
                            className={selectElementClassName}
                            wrapperClass={selectWrapperClassName}
                            triggerClass="border-l border-base-300"
                            value={selectedCluster.type}
                        />
                    </div>
                    <div className="mb-4">
                        <label htmlFor="mainImage" className={labelClassName}>
                            Main Image Repository{' '}
                            <FormFieldRequired empty={selectedCluster.mainImage.length === 0} />
                        </label>
                        <input
                            id="mainImage"
                            name="mainImage"
                            onChange={handleChange}
                            value={selectedCluster.mainImage}
                            className={inputTextClassName}
                        />
                    </div>
                    <div className="mb-4">
                        <label htmlFor="centralApiEndpoint" className={labelClassName}>
                            Central API Endpoint (include port){' '}
                            <FormFieldRequired
                                empty={selectedCluster.centralApiEndpoint.length === 0}
                            />
                        </label>
                        <input
                            id="centralApiEndpoint"
                            name="centralApiEndpoint"
                            onChange={handleChange}
                            value={selectedCluster.centralApiEndpoint}
                            className={inputTextClassName}
                        />
                    </div>
                    <div className="mb-4">
                        <label htmlFor="collectionMethod" className={labelClassName}>
                            Collection Method
                        </label>
                        <Select
                            options={runtimeOptions}
                            placeholder="Select a runtime option"
                            onChange={onCollectionMethodChange}
                            className={selectElementClassName}
                            wrapperClass={selectWrapperClassName}
                            triggerClass="border-l border-base-300"
                            value={selectedCluster.collectionMethod}
                        />
                    </div>
                    <div className="mb-4">
                        <label htmlFor="collectorImage" className={labelClassName}>
                            Collector Image Repository (uses Main image repository by default)
                        </label>
                        <input
                            id="collectorImage"
                            name="collectorImage"
                            onChange={handleChange}
                            value={selectedCluster.collectorImage}
                            className={inputTextClassName}
                        />
                    </div>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="admissionController" className={labelClassName}>
                            Create Admission Controller Webhook
                        </label>
                        <ToggleSwitch
                            id="admissionController"
                            name="admissionController"
                            toggleHandler={handleChange}
                            enabled={selectedCluster.admissionController}
                        />
                    </div>
                    <FeatureEnabled
                        featureFlag={knownBackendFlags.ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE}
                    >
                        {({ featureEnabled }) => {
                            return (
                                featureEnabled && (
                                    <div
                                        className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}
                                    >
                                        <label
                                            htmlFor="admissionControllerUpdates"
                                            className={labelClassName}
                                        >
                                            Configure Admission Controller Webhook to listen on
                                            updates
                                        </label>
                                        <ToggleSwitch
                                            id="admissionControllerUpdates"
                                            name="admissionControllerUpdates"
                                            toggleHandler={handleChange}
                                            enabled={
                                                selectedCluster.admissionController &&
                                                selectedCluster.admissionControllerUpdates
                                            }
                                            disabled={!selectedCluster.admissionController}
                                        />
                                    </div>
                                )
                            );
                        }}
                    </FeatureEnabled>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="tolerationsConfig.disabled" className={labelClassName}>
                            <span>Enable Taint Tolerations</span>
                            <br />
                            <span className={sublabelClassName}>
                                Tolerate all taints to run on all nodes of this cluster
                            </span>
                        </label>
                        <ToggleSwitch
                            id="tolerationsConfig.disabled"
                            name="tolerationsConfig.disabled"
                            toggleHandler={handleChange}
                            flipped
                            // TODO: check until API guarantees a tolerationsConfig object is returned
                            // with false, if not yet set
                            enabled={
                                !(
                                    selectedCluster.tolerationsConfig === null ||
                                    selectedCluster.tolerationsConfig.disabled === false
                                )
                            }
                        />
                    </div>
                    <FeatureEnabled featureFlag={knownBackendFlags.ROX_SUPPORT_SLIM_COLLECTOR_MODE}>
                        {({ featureEnabled }) => {
                            return (
                                featureEnabled && (
                                    <div className={`flex flex-col ${divToggleOuterClassName}`}>
                                        <div className={justifyBetweenClassName}>
                                            <label
                                                htmlFor="slimCollector"
                                                className={labelClassName}
                                            >
                                                <span>Enable Slim Collector Mode</span>
                                                <br />
                                                <span className={sublabelClassName}>
                                                    New cluster will be set up using a slim
                                                    collector image
                                                </span>
                                            </label>
                                            <ToggleSwitch
                                                id="slimCollector"
                                                name="slimCollector"
                                                toggleHandler={handleChange}
                                                enabled={selectedCluster.slimCollector}
                                            />
                                        </div>
                                        {renderSlimCollectorWarning()}
                                    </div>
                                )
                            );
                        }}
                    </FeatureEnabled>
                </div>
            </CollapsibleCard>
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            <CollapsibleCard
                title="Dynamic Configuration (syncs with Sensor)"
                titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
            >
                <div className="p-3">
                    <div className="mb-4">
                        <label htmlFor="dynamicConfig.registryOverride" className={labelClassName}>
                            <span>Custom default image registry</span>
                            <br />
                            <span className={sublabelClassName}>
                                Set a value if the default registry is not docker.io in this cluster
                            </span>
                        </label>
                        <div className="flex">
                            <input
                                id="dynamicConfig.registryOverride"
                                name="dynamicConfig.registryOverride"
                                onChange={handleChange}
                                value={selectedCluster.dynamicConfig.registryOverride}
                                className={inputTextClassName}
                                placeholder="image-mirror.example.com"
                            />
                        </div>
                    </div>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.enabled"
                            className={labelClassName}
                        >
                            Enable Admission Controller
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.admissionControllerConfig.enabled"
                            name="dynamicConfig.admissionControllerConfig.enabled"
                            toggleHandler={handleChange}
                            enabled={
                                selectedCluster.dynamicConfig.admissionControllerConfig.enabled
                            }
                        />
                    </div>
                    <FeatureEnabled
                        featureFlag={knownBackendFlags.ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE}
                    >
                        {({ featureEnabled }) => {
                            return (
                                featureEnabled && (
                                    <div
                                        className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}
                                    >
                                        <label
                                            htmlFor="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                                            className={labelClassName}
                                        >
                                            Enforce on Updates
                                        </label>
                                        <ToggleSwitch
                                            id="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                                            name="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                                            toggleHandler={handleChange}
                                            enabled={
                                                selectedCluster.dynamicConfig
                                                    .admissionControllerConfig.enabled &&
                                                selectedCluster.dynamicConfig
                                                    .admissionControllerConfig.enforceOnUpdates
                                            }
                                            disabled={
                                                !selectedCluster.dynamicConfig
                                                    .admissionControllerConfig.enabled
                                            }
                                        />
                                    </div>
                                )
                            );
                        }}
                    </FeatureEnabled>
                    <div className={`mb-4 pl-2 ${justifyBetweenClassName}`}>
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig
                        .timeoutSeconds"
                            className={labelClassName}
                        >
                            Timeout (seconds)
                        </label>
                        <input
                            className={inputNumberClassName}
                            id="dynamicConfig.admissionControllerConfig.timeoutSeconds"
                            name="dynamicConfig.admissionControllerConfig.timeoutSeconds"
                            onChange={handleChange}
                            value={
                                selectedCluster.dynamicConfig.admissionControllerConfig
                                    .timeoutSeconds
                            }
                        />
                    </div>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.scanInline"
                            className={labelClassName}
                        >
                            Contact Image Scanners
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.admissionControllerConfig.scanInline"
                            name="dynamicConfig.admissionControllerConfig.scanInline"
                            toggleHandler={handleChange}
                            enabled={
                                selectedCluster.dynamicConfig.admissionControllerConfig.scanInline
                            }
                        />
                    </div>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.disableBypass"
                            className={labelClassName}
                        >
                            Disable Use of Bypass Annotation
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.admissionControllerConfig.disableBypass"
                            name="dynamicConfig.admissionControllerConfig.disableBypass"
                            toggleHandler={handleChange}
                            enabled={
                                selectedCluster.dynamicConfig.admissionControllerConfig
                                    .disableBypass
                            }
                        />
                    </div>
                </div>
            </CollapsibleCard>
        </form>
    );
}

ClusterEditForm.propTypes = {
    centralEnv: PropTypes.shape({
        kernelSupportAvailable: PropTypes.bool,
    }).isRequired,
    centralVersion: PropTypes.string.isRequired,
    selectedCluster: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string,
        type: PropTypes.string,
        mainImage: PropTypes.string,
        centralApiEndpoint: PropTypes.string,
        collectionMethod: PropTypes.string,
        collectorImage: PropTypes.string,
        admissionController: PropTypes.bool,
        admissionControllerUpdates: PropTypes.bool,
        tolerationsConfig: PropTypes.shape({
            disabled: PropTypes.bool,
        }),
        status: PropTypes.object,
        dynamicConfig: PropTypes.shape({
            registryOverride: PropTypes.string,
            admissionControllerConfig: PropTypes.shape({
                enabled: PropTypes.bool,
                enforceOnUpdates: PropTypes.bool,
                timeoutSeconds: PropTypes.number,
                scanInline: PropTypes.bool,
                disableBypass: PropTypes.bool,
            }),
        }),
        slimCollector: PropTypes.bool,
        healthStatus: PropTypes.object,
    }).isRequired,
    handleChange: PropTypes.func.isRequired,
    isLoading: PropTypes.bool.isRequired,
};

export default ClusterEditForm;
