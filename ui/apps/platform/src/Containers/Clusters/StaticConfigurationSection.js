import React from 'react';
import { Message } from '@stackrox/ui-components';

import CollapsibleSection from 'Components/CollapsibleSection';
import ToggleSwitch from 'Components/ToggleSwitch';
import FormFieldRequired from 'Components/forms/FormFieldRequired';
import Select from 'Components/Select';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';

import {
    clusterTypeOptions,
    runtimeOptions,
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
    divToggleOuterClassName,
    justifyBetweenClassName,
    selectElementClassName,
    selectWrapperClassName,
} from './cluster.helpers';
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

const StaticConfigurationSection = ({ centralEnv, selectedCluster, handleChange }) => {
    const slimCollectorEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_SUPPORT_SLIM_COLLECTOR_MODE
    );
    const k8sEventsEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_EVENTS_DETECTION);

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

    const showSlimCollectorWarning =
        centralEnv?.successfullyFetched &&
        selectedCluster.slimCollector &&
        !centralEnv.kernelSupportAvailable;

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
                        onChange={onClusterTypeChange}
                        className={selectElementClassName}
                        wrapperClass={selectWrapperClassName}
                        triggerClass="border-l border-base-300"
                        value={selectedCluster.type}
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
                        />
                    </label>
                    <div data-testid="input-wrapper">
                        <input
                            id="centralApiEndpoint"
                            name="centralApiEndpoint"
                            onChange={handleChange}
                            value={selectedCluster.centralApiEndpoint}
                            className={inputTextClassName}
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
                        />
                        <HelmValueWarning
                            currentValue={selectedCluster.collectorImage}
                            helmValue={selectedCluster?.helmConfig?.staticConfig?.collectorImage}
                        />
                    </div>
                </div>
                {k8sEventsEnabled && (
                    <div className={wrapperMarginClassName}>
                        <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                            <label htmlFor="admissionControllerEvents" className={labelClassName}>
                                Enable Admission Controller Webhook to listen on exec and
                                port-forward events
                            </label>
                            <ToggleSwitch
                                id="admissionControllerEvents"
                                name="admissionControllerEvents"
                                toggleHandler={handleChange}
                                enabled={selectedCluster.admissionControllerEvents}
                            />
                        </div>
                        <HelmValueWarning
                            currentValue={selectedCluster.admissionControllerEvents}
                            helmValue={
                                selectedCluster?.helmConfig?.staticConfig?.admissionControllerEvents
                            }
                        />
                        {selectedCluster.admissionControllerEvents &&
                            selectedCluster?.type === 'OPENSHIFT_CLUSTER' && (
                                <div className="border border-alert-200 bg-alert-200 p-2 rounded-b">
                                    This setting will not work for OpenShift 3.11; so please ensure
                                    that your cluster is running OpenShift 4.0 and higher.
                                </div>
                            )}
                    </div>
                )}
                <div className={wrapperMarginClassName}>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="admissionController" className={labelClassName}>
                            {k8sEventsEnabled
                                ? 'Configure Admission Controller Webhook to listen on Object Creates'
                                : 'Create Admission Controller Webhook'}
                        </label>
                        <ToggleSwitch
                            id="admissionController"
                            name="admissionController"
                            toggleHandler={handleChange}
                            enabled={selectedCluster.admissionController}
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
                            {k8sEventsEnabled
                                ? 'Configure Admission Controller Webhook to listen on Object Updates'
                                : 'Configure Admission Controller Webhook to listen on updates'}
                        </label>
                        <ToggleSwitch
                            id="admissionControllerUpdates"
                            name="admissionControllerUpdates"
                            toggleHandler={handleChange}
                            enabled={selectedCluster.admissionControllerUpdates}
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
                {slimCollectorEnabled && (
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
                            <Message type="warn">
                                Failed to check if Central has kernel support packages available
                            </Message>
                        )}
                        {showSlimCollectorWarning && (
                            <Message type="warn">
                                <span>
                                    Central doesn’t have the required Kernel support package.
                                    Retrieve it from{' '}
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
                            </Message>
                        )}
                    </div>
                )}
            </div>
        </CollapsibleSection>
    );
};

export default StaticConfigurationSection;
