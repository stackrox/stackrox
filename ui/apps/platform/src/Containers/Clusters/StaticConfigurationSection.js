import React from 'react';
import { Alert } from '@patternfly/react-core';

import CollapsibleSection from 'Components/CollapsibleSection';
import ToggleSwitch from 'Components/ToggleSwitch';
import Select from 'Components/Select';
import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
    divToggleOuterClassName,
    justifyBetweenClassName,
    selectElementClassName,
    selectWrapperClassName,
} from 'constants/form.constants';

import { clusterTypeOptions, clusterTypes, runtimeOptions } from './cluster.helpers';
import FormFieldRequired from './Components/FormFieldRequired';
import HelmValueWarning from './Components/HelmValueWarning';

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

const StaticConfigurationSection = ({
    centralEnv,
    selectedCluster,
    isManagerTypeNonConfigurable,
    handleChange,
}) => {
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
    function guardedClusterTypeChange(selectedOption) {
        if (selectedOption.value === clusterTypes.OPENSHIFT_3) {
            // force admission controller events off in OpenShift 3.x
            const syntheticEvent = {
                target: {
                    name: 'admissionControllerEvents',
                    value: false,
                },
            };

            handleChange(syntheticEvent);
        }
        onClusterTypeChange(selectedOption);
    }

    const showSlimCollectorWarning =
        centralEnv?.successfullyFetched &&
        selectedCluster.slimCollector &&
        !centralEnv.kernelSupportAvailable;

    const isTypeOpenShift3 = selectedCluster?.type === clusterTypes.OPENSHIFT_3;

    return (
        <CollapsibleSection
            title="Static Configuration (requires deployment)"
            titleClassName="text-xl"
        >
            <div className="bg-base-100 pb-3 pt-1 px-3 rounded shadow">
                <div className={wrapperMarginClassName}>
                    <label htmlFor="name" className={labelClassName}>
                        Cluster Name <FormFieldRequired empty={selectedCluster.name.length === 0} />
                    </label>
                    <div data-testid="input-wrapper">
                        <input
                            id="name"
                            name="name"
                            value={selectedCluster.name}
                            onChange={handleChange}
                            disabled={selectedCluster.id}
                            className={inputTextClassName}
                        />
                    </div>
                </div>
                <div className={wrapperMarginClassName}>
                    <label htmlFor="clusterType" className={labelClassName}>
                        Cluster Type <FormFieldRequired empty={selectedCluster.type.length === 0} />
                    </label>
                    <Select
                        id="clusterType"
                        options={clusterTypeOptions}
                        placeholder="Select a cluster type"
                        onChange={guardedClusterTypeChange}
                        className={selectElementClassName}
                        wrapperClass={selectWrapperClassName}
                        triggerClass="border-l border-base-300"
                        value={selectedCluster.type}
                        disabled={isManagerTypeNonConfigurable}
                    />
                    <HelmValueWarning
                        currentValue={selectedCluster.type}
                        helmValue={selectedCluster?.helmConfig?.staticConfig?.type}
                    />
                </div>
                <div className={wrapperMarginClassName}>
                    <label htmlFor="mainImage" className={labelClassName}>
                        Main Image Repository{' '}
                        <FormFieldRequired empty={selectedCluster.mainImage.length === 0} />
                    </label>
                    <div data-testid="input-wrapper">
                        <input
                            id="mainImage"
                            name="mainImage"
                            onChange={handleChange}
                            value={selectedCluster.mainImage}
                            className={inputTextClassName}
                            disabled={isManagerTypeNonConfigurable}
                        />
                        <HelmValueWarning
                            currentValue={selectedCluster.mainImage}
                            helmValue={selectedCluster?.helmConfig?.staticConfig?.mainImage}
                        />
                    </div>
                </div>
                <div className={wrapperMarginClassName}>
                    <label htmlFor="centralApiEndpoint" className={labelClassName}>
                        Central API Endpoint (include port){' '}
                        <FormFieldRequired
                            empty={selectedCluster.centralApiEndpoint.length === 0}
                            disabled={isManagerTypeNonConfigurable}
                        />
                    </label>
                    <div data-testid="input-wrapper">
                        <input
                            id="centralApiEndpoint"
                            name="centralApiEndpoint"
                            onChange={handleChange}
                            value={selectedCluster.centralApiEndpoint}
                            className={inputTextClassName}
                            disabled={isManagerTypeNonConfigurable}
                        />
                        <HelmValueWarning
                            currentValue={selectedCluster.centralApiEndpoint}
                            helmValue={
                                selectedCluster?.helmConfig?.staticConfig?.centralApiEndpoint
                            }
                        />
                    </div>
                </div>
                <div className={wrapperMarginClassName}>
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
                        disabled={isManagerTypeNonConfigurable}
                    />
                    <HelmValueWarning
                        currentValue={selectedCluster.collectionMethod}
                        helmValue={selectedCluster?.helmConfig?.staticConfig?.collectionMethod}
                    />
                </div>
                <div className={wrapperMarginClassName}>
                    <label htmlFor="collectorImage" className={labelClassName}>
                        Collector Image Repository (uses Main image repository by default)
                    </label>
                    <div data-testid="input-wrapper">
                        <input
                            id="collectorImage"
                            name="collectorImage"
                            onChange={handleChange}
                            value={selectedCluster.collectorImage}
                            className={inputTextClassName}
                            disabled={isManagerTypeNonConfigurable}
                        />
                        <HelmValueWarning
                            currentValue={selectedCluster.collectorImage}
                            helmValue={selectedCluster?.helmConfig?.staticConfig?.collectorImage}
                        />
                    </div>
                </div>
                <div className={wrapperMarginClassName}>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="admissionControllerEvents" className={labelClassName}>
                            Enable Admission Controller Webhook to listen on exec and port-forward
                            events
                        </label>
                        <ToggleSwitch
                            id="admissionControllerEvents"
                            name="admissionControllerEvents"
                            disabled={isTypeOpenShift3 || isManagerTypeNonConfigurable}
                            toggleHandler={handleChange}
                            enabled={
                                isTypeOpenShift3 ? false : selectedCluster.admissionControllerEvents
                            }
                        />
                    </div>
                    {!isTypeOpenShift3 && (
                        <HelmValueWarning
                            currentValue={selectedCluster.admissionControllerEvents}
                            helmValue={
                                selectedCluster?.helmConfig?.staticConfig?.admissionControllerEvents
                            }
                        />
                    )}
                    {isTypeOpenShift3 && (
                        <div className="border border-alert-200 bg-alert-200 p-2 rounded-b">
                            This setting will not work for OpenShift 3.11. To use this webhook, you
                            must upgrade your cluster to OpenShift 4.1 or higher.
                        </div>
                    )}
                </div>
                <div className={wrapperMarginClassName}>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="admissionController" className={labelClassName}>
                            Configure Admission Controller Webhook to listen on Object Creates
                        </label>
                        <ToggleSwitch
                            id="admissionController"
                            name="admissionController"
                            toggleHandler={handleChange}
                            enabled={selectedCluster.admissionController}
                            disabled={isManagerTypeNonConfigurable}
                        />
                    </div>
                    <HelmValueWarning
                        currentValue={selectedCluster.admissionController}
                        helmValue={selectedCluster?.helmConfig?.staticConfig?.admissionController}
                    />
                </div>
                <div className={wrapperMarginClassName}>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="admissionControllerUpdates" className={labelClassName}>
                            Configure Admission Controller Webhook to listen on Object Updates
                        </label>
                        <ToggleSwitch
                            id="admissionControllerUpdates"
                            name="admissionControllerUpdates"
                            toggleHandler={handleChange}
                            enabled={selectedCluster.admissionControllerUpdates}
                            disabled={isManagerTypeNonConfigurable}
                        />
                    </div>
                    <HelmValueWarning
                        currentValue={selectedCluster.admissionControllerUpdates}
                        helmValue={
                            selectedCluster?.helmConfig?.staticConfig?.admissionControllerUpdates
                        }
                    />
                </div>
                <div className={wrapperMarginClassName}>
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
                            disabled={isManagerTypeNonConfigurable}
                            enabled={
                                !(
                                    selectedCluster.tolerationsConfig === null ||
                                    selectedCluster.tolerationsConfig.disabled === false
                                )
                            }
                        />
                    </div>
                    <HelmValueWarning
                        currentValue={selectedCluster?.tolerationsConfig?.disabled}
                        helmValue={
                            selectedCluster?.helmConfig?.staticConfig?.tolerationsConfig?.disabled
                        }
                    />
                </div>
                <div className="flex flex-col">
                    <div className={wrapperMarginClassName}>
                        <div className={divToggleOuterClassName}>
                            <div className={justifyBetweenClassName}>
                                <label htmlFor="slimCollector" className={labelClassName}>
                                    <span>Enable Slim Collector Mode</span>
                                    <br />
                                    <span className={sublabelClassName}>
                                        New cluster will be set up using a slim collector image
                                    </span>
                                </label>
                                <ToggleSwitch
                                    id="slimCollector"
                                    name="slimCollector"
                                    toggleHandler={handleChange}
                                    disabled={isManagerTypeNonConfigurable}
                                    enabled={selectedCluster.slimCollector}
                                />
                            </div>
                        </div>
                        <HelmValueWarning
                            currentValue={selectedCluster?.slimCollector}
                            helmValue={selectedCluster?.helmConfig?.staticConfig?.slimCollector}
                        />
                    </div>
                    {!centralEnv?.successfullyFetched && (
                        <Alert
                            variant="warning"
                            isInline
                            title="Failed to check if Central has kernel support packages available"
                        />
                    )}
                    {showSlimCollectorWarning && (
                        <Alert variant="warning" isInline title="Kernel support package">
                            <span>
                                Central doesn’t have the required Kernel support package. Retrieve
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
                        </Alert>
                    )}
                </div>
            </div>
        </CollapsibleSection>
    );
};

export default StaticConfigurationSection;
