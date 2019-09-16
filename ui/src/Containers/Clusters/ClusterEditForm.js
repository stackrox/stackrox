import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';
import Select from 'Components/Select';
import ToggleSwitch from 'Components/ToggleSwitch';
import { clusterTypeOptions, runtimeOptions } from './cluster.helpers';

// factory that returns a handler to normalize our generic Select component's return value
function getSelectComparison(options, key, selectedCluster, handleChange) {
    return function compareSelected(selectedOption) {
        if (
            options.find(value => value === selectedOption) !== undefined &&
            selectedCluster[key] !== selectedOption.value
        ) {
            const syntheticEvent = {
                target: {
                    name: key,
                    value: selectedOption.value
                }
            };

            handleChange(syntheticEvent);
        }
    };
}

function ClusterEditForm({ selectedCluster, handleChange }) {
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

    return (
        <form className="px-4 w-full mb-8">
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            <CollapsibleCard
                open
                title="Static Configuration (requires deployment)"
                cardClassName="border border-base-400 mb-2"
                titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
            >
                <div className="p-3">
                    <div className="mb-4">
                        <label htmlFor="name" className="block py-2 text-base-600 font-700">
                            Cluster Name{' '}
                            <span
                                aria-label="Required"
                                data-test-id="required"
                                className="text-alert-500 ml-1"
                            >
                                *
                            </span>
                        </label>
                        <div className="flex">
                            <input
                                id="name"
                                name="name"
                                value={selectedCluster.name}
                                onChange={handleChange}
                                disabled={selectedCluster.id}
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="mainImage" className="block py-2 text-base-600 font-700">
                            Main Image Repository{' '}
                            <span
                                aria-label="Required"
                                data-test-id="required"
                                className="text-alert-500 ml-1"
                            >
                                *
                            </span>
                        </label>
                        <div className="flex">
                            <input
                                id="mainImage"
                                name="mainImage"
                                onChange={handleChange}
                                value={selectedCluster.mainImage}
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label
                            htmlFor="centralApiEndpoint"
                            className="block py-2 text-base-600 font-700"
                        >
                            Central API Endpoint (include port){' '}
                            <span
                                aria-label="Required"
                                data-test-id="required"
                                className="text-alert-500 ml-1"
                            >
                                *
                            </span>
                        </label>
                        <div className="flex">
                            <input
                                id="centralApiEndpoint"
                                name="centralApiEndpoint"
                                onChange={handleChange}
                                value={selectedCluster.centralApiEndpoint}
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label
                            htmlFor="monitoringEndpoint"
                            className="block py-2 text-base-600 font-700"
                        >
                            Monitoring Endpoint (include port; empty means no monitoring)
                        </label>
                        <div className="flex">
                            <input
                                id="monitoringEndpoint"
                                name="monitoringEndpoint"
                                onChange={handleChange}
                                value={selectedCluster.monitoringEndpoint}
                                placeholder="<monitoring-subdomain>.<domain>:<port>"
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label
                            htmlFor="collectionMethod"
                            className="block py-2 text-base-600 font-700"
                        >
                            Collection Method
                        </label>
                        <div className="flex">
                            <Select
                                options={runtimeOptions}
                                placeholder="Select a runtime option"
                                onChange={onCollectionMethodChange}
                                className="block w-full bg-base-100 border-base-300 text-base-600 p-3 pr-8 z-1 focus:border-base-500"
                                wrapperClass="bg-base-100 border-2 rounded border-base-300 w-full font-600 text-base-600 hover:border-base-400"
                                triggerClass="border-l border-base-300"
                                value={selectedCluster.collectionMethod}
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label
                            htmlFor="collectorImage"
                            className="block py-2 text-base-600 font-700"
                        >
                            Collector Image Repository (uses Main image repository by default)
                        </label>
                        <div className="flex">
                            <input
                                id="collectorImage"
                                name="collectorImage"
                                onChange={handleChange}
                                value={selectedCluster.collectorImage}
                                className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                            />
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="clusterType" className="block py-2 text-base-600 font-700">
                            Cluster Type
                        </label>
                        <div className="flex">
                            <Select
                                id="clusterType"
                                options={clusterTypeOptions}
                                placeholder="Select a cluster type"
                                onChange={onClusterTypeChange}
                                className="block w-full border-r bg-base-100 border-base-300 text-base-600 p-3 pr-8 z-1 focus:border-base-500"
                                wrapperClass="bg-base-100 border-2 rounded border-base-300 w-full font-600 text-base-600 hover:border-base-400"
                                triggerClass="border-l border-base-300"
                                value={selectedCluster.type}
                            />
                        </div>
                    </div>
                    {selectedCluster.type === 'KUBERNETES_CLUSTER' && (
                        <div className="mb-4 flex bg-base-100 border-2 rounded px-2 py-1 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10 border-base-300 items-center justify-between">
                            <label
                                htmlFor="admissionController"
                                className="block py-2 text-base-600 font-700"
                            >
                                Create Admission Controller Webhook
                            </label>
                            <ToggleSwitch
                                id="admissionController"
                                name="admissionController"
                                toggleHandler={handleChange}
                                enabled={selectedCluster.admissionController}
                            />
                        </div>
                    )}
                </div>
            </CollapsibleCard>
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            {selectedCluster.type === 'KUBERNETES_CLUSTER' && (
                <CollapsibleCard
                    open={false}
                    title="Dynamic Configuration (syncs with Sensor)"
                    titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
                >
                    <div className="p-3">
                        <h3>Admission Controller</h3>
                        <div className="mb-4 flex bg-base-100 border-2 rounded px-2 py-1 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10 border-base-300 items-center justify-between">
                            <label
                                htmlFor="dynamicConfig.admissionControllerConfig.enabled"
                                className="block py-2 text-base-600 font-700"
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
                        <div className="mb-4 flex px-2 py-2 items-center justify-between">
                            <label
                                htmlFor="dynamicConfig.admissionControllerConfig
                            .timeoutSeconds"
                                className="py-2 text-base-600 font-700 flex"
                            >
                                Timeout (Seconds)
                            </label>
                            <input
                                className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                                id="dynamicConfig.admissionControllerConfig.timeoutSeconds"
                                name="dynamicConfig.admissionControllerConfig.timeoutSeconds"
                                onChange={handleChange}
                                value={
                                    selectedCluster.dynamicConfig.admissionControllerConfig
                                        .timeoutSeconds
                                }
                            />
                        </div>
                        <div className="mb-4 flex bg-base-100 border-2 rounded px-2 py-1 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10 border-base-300 items-center justify-between">
                            <label
                                htmlFor="dynamicConfig.admissionControllerConfig.scanInline"
                                className="block py-2 text-base-600 font-700"
                            >
                                Contact Image Scanners
                            </label>
                            <ToggleSwitch
                                id="dynamicConfig.admissionControllerConfig.scanInline"
                                name="dynamicConfig.admissionControllerConfig.scanInline"
                                toggleHandler={handleChange}
                                enabled={
                                    selectedCluster.dynamicConfig.admissionControllerConfig
                                        .scanInline
                                }
                            />
                        </div>
                        <div className="mb-4 flex bg-base-100 border-2 rounded px-2 py-1 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10 border-base-300 items-center justify-between">
                            <label
                                htmlFor="dynamicConfig.admissionControllerConfig.disableBypass"
                                className="block py-2 text-base-600 font-700"
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
            )}
        </form>
    );
}

ClusterEditForm.propTypes = {
    selectedCluster: PropTypes.shape({}).isRequired,
    handleChange: PropTypes.func.isRequired
};

export default ClusterEditForm;
