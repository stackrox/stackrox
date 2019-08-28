import React from 'react';
import PropTypes from 'prop-types';
import CollapsibleCard from 'Components/CollapsibleCard';

function ClusterEditForm({ selectedCluster, handleChange }) {
    // @TODO, only show certain fields based on cluster type

    return (
        <form className="p-4 w-full mb-8">
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            <CollapsibleCard
                open={false}
                title="Static Configuration (requires deployment)"
                cardClassName="border border-base-400 mb-2"
                titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
            >
                <div className="p-3">
                    <div className="mb-4">
                        <label htmlFor="name" className="py-2 text-base-600 font-700">
                            Name:
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="name"
                                    name="name"
                                    value={selectedCluster.name}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="mainImage" className="py-2 text-base-600 font-700">
                            Main Image Repository:
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="mainImage"
                                    name="mainImage"
                                    onChange={handleChange}
                                    value={selectedCluster.mainImage}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="centralApiEndpoint" className="py-2 text-base-600 font-700">
                            Central API Endpoint (include port):
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="centralApiEndpoint"
                                    name="centralApiEndpoint"
                                    onChange={handleChange}
                                    value={selectedCluster.centralApiEndpoint}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="monitoringEndpoint" className="py-2 text-base-600 font-700">
                            Monitoring Endpoint (include port; empty means no monitoring):
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="monitoringEndpoint"
                                    name="monitoringEndpoint"
                                    onChange={handleChange}
                                    value={selectedCluster.monitoringEndpoint}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="collectionMethod" className="py-2 text-base-600 font-700">
                            Collection Method:
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <select
                                    id="collectionMethod"
                                    name="collectionMethod"
                                    onChange={handleChange}
                                    value={selectedCluster.collectionMethod}
                                >
                                    <option value="NO_COLLECTION">No Runtime Support</option>
                                    <option value="KERNEL_MODULE">Kernel Module Support</option>
                                    <option value="EBPF">eBPF Support</option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label htmlFor="collectorImage" className="py-2 text-base-600 font-700">
                            Collector Image Repository (uses Main image repository by default):
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="collectorImage"
                                    name="collectorImage"
                                    onChange={handleChange}
                                    value={selectedCluster.collectorImage}
                                />
                            </div>
                        </div>
                    </div>
                    <div className="mb-4">
                        <label
                            htmlFor="admissionController"
                            className="py-2 text-base-600 font-700"
                        >
                            Create Admission Controller Webhook:
                        </label>
                        <div className="flex">
                            <div className="bg-base-100 border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 hover:border-base-400 leading-normal min-h-10">
                                <input
                                    id="admissionController"
                                    name="admissionController"
                                    onChange={handleChange}
                                    type="checkbox"
                                    value={selectedCluster.admissionController}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </CollapsibleCard>
            {/* @TODO, replace open prop with dynamic logic, based on clusterType */}
            <CollapsibleCard
                open
                title="Dynamic Configuration (syncs with Sensor)"
                titleClassName="border-b border-base-300 bg-primary-200 leading-normal cursor-pointer flex justify-between items-center hover:bg-primary-300 hover:border-primary-300"
            >
                <div className="p-3">
                    <h3>Admission Controller</h3>
                    <div className="mb-4 flex py-2 border-b-2 border-base-300 items-center justify-between">
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.enabled"
                            className="py-2 text-base-600 font-700 flex w-full"
                        >
                            Enable Admission Controller:
                        </label>
                        <div className="">
                            <input
                                className="w-12"
                                id="dynamicConfig.admissionControllerConfig.enabled"
                                name="dynamicConfig.admissionControllerConfig.enabled"
                                onChange={handleChange}
                                type="checkbox"
                                value={
                                    selectedCluster.dynamicConfig.admissionControllerConfig.enabled
                                }
                            />
                        </div>
                    </div>
                    <div className="mb-4 flex py-2 border-b-2 border-base-300 items-center justify-between">
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig
                            .timeoutSeconds"
                            className="py-2 text-base-600 font-700 flex"
                        >
                            Timeout (Seconds):
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
                    <div className="mb-4 flex py-2 border-b-2 border-base-300 items-center justify-between">
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.scanInline"
                            className="py-2 text-base-600 font-700 flex w-full"
                        >
                            Contact Image Scanners:
                        </label>
                        <div className="">
                            <input
                                className="w-12"
                                id="dynamicConfig.admissionControllerConfig.scanInline"
                                name="dynamicConfig.admissionControllerConfig.scanInline"
                                onChange={handleChange}
                                type="checkbox"
                                value={
                                    selectedCluster.dynamicConfig.admissionControllerConfig
                                        .scanInline
                                }
                            />
                        </div>
                    </div>
                    <div className="mb-4 flex py-2 border-b-2 border-base-300">
                        <label
                            htmlFor="dynamicConfig.admissionControllerConfig.disableBypass"
                            className="py-2 text-base-600 font-700 flex w-full"
                        >
                            Disable Use of Bypass Annotation:
                        </label>
                        <div className="">
                            <input
                                className="w-12"
                                id="dynamicConfig.admissionControllerConfig.disableBypass"
                                name="dynamicConfig.admissionControllerConfig.disableBypass"
                                onChange={handleChange}
                                type="checkbox"
                                value={
                                    selectedCluster.dynamicConfig.admissionControllerConfig
                                        .disableBypass
                                }
                            />
                        </div>
                    </div>
                </div>
            </CollapsibleCard>
        </form>
    );
}

ClusterEditForm.propTypes = {
    selectedCluster: PropTypes.shape({}).isRequired,
    handleChange: PropTypes.func.isRequired
};

export default ClusterEditForm;
