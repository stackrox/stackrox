import React from 'react';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import CollapsibleSection from 'Components/CollapsibleSection';
import ToggleSwitch from 'Components/ToggleSwitch';

const labelClassName = 'block py-2 text-base-600 font-700';
const sublabelClassName = 'font-600 italic';

const inputBaseClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal p-2 rounded text-base-600';
const inputTextClassName = `${inputBaseClassName} w-full`;
const inputNumberClassName = `${inputBaseClassName} text-right w-12`;

const divToggleOuterClassName =
    'bg-base-100 border-2 border-base-300 hover:border-base-400 font-600 leading-normal mb-4 px-2 py-2 rounded text-base-600 w-full';

const justifyBetweenClassName = 'flex items-center justify-between';

const DynamicConfigurationSection = ({ handleChange, dynamicConfig }) => {
    const k8sEventsEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_EVENTS_DETECTION);
    const { registryOverride, admissionControllerConfig } = dynamicConfig;
    // @TODO, replace open prop with dynamic logic, based on clusterType
    return (
        <CollapsibleSection
            title="Dynamic Configuration (syncs with Sensor)"
            titleClassName="text-xl"
        >
            <div className="bg-base-100 pb-3 pt-1 px-3 rounded shadow">
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
                            value={registryOverride}
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
                        {k8sEventsEnabled
                            ? 'Enforce on Object Creates'
                            : 'Enable Admission Controller'}
                    </label>
                    <ToggleSwitch
                        id="dynamicConfig.admissionControllerConfig.enabled"
                        name="dynamicConfig.admissionControllerConfig.enabled"
                        toggleHandler={handleChange}
                        enabled={admissionControllerConfig.enabled}
                    />
                </div>
                <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                    <label
                        htmlFor="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                        className={labelClassName}
                    >
                        {k8sEventsEnabled ? 'Enforce on Object Updates' : 'Enforce on Updates'}
                    </label>
                    <ToggleSwitch
                        id="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                        name="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                        toggleHandler={handleChange}
                        enabled={admissionControllerConfig.enforceOnUpdates}
                    />
                </div>
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
                        value={admissionControllerConfig.timeoutSeconds}
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
                        enabled={admissionControllerConfig.scanInline}
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
                        enabled={admissionControllerConfig.disableBypass}
                    />
                </div>
            </div>
        </CollapsibleSection>
    );
};

export default DynamicConfigurationSection;
