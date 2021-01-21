import React from 'react';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import CollapsibleSection from 'Components/CollapsibleSection';
import ToggleSwitch from 'Components/ToggleSwitch';

import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
    inputNumberClassName,
    divToggleOuterClassName,
    justifyBetweenClassName,
} from './cluster.helpers';
import HelmValueWarning from './Components/HelmValueWarning';

const DynamicConfigurationSection = ({ handleChange, dynamicConfig, helmConfig }) => {
    const k8sEventsEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_EVENTS_DETECTION);
    const { registryOverride, admissionControllerConfig } = dynamicConfig;
    // @TODO, replace open prop with dynamic logic, based on clusterType
    return (
        <CollapsibleSection
            title="Dynamic Configuration (syncs with Sensor)"
            titleClassName="text-xl"
        >
            <div className="bg-base-100 pb-3 pt-1 px-3 rounded shadow">
                <div className={wrapperMarginClassName}>
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
                    <HelmValueWarning
                        currentValue={dynamicConfig.registryOverride}
                        helmValue={helmConfig?.dynamicConfig?.registryOverride}
                    />
                </div>
                <div className={wrapperMarginClassName}>
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
                    {helmConfig &&
                        dynamicConfig.admissionControllerConfig.enabled !==
                            helmConfig?.dynamicConfig?.admissionControllerConfig?.enabled && (
                            <HelmValueWarning
                                currentValue={dynamicConfig.admissionControllerConfig.enabled}
                                helmValue={
                                    helmConfig?.dynamicConfig?.admissionControllerConfig?.enabled
                                }
                            />
                        )}
                </div>
                <div className={wrapperMarginClassName}>
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
                    <HelmValueWarning
                        currentValue={dynamicConfig.admissionControllerConfig.enforceOnUpdates}
                        helmValue={
                            helmConfig?.dynamicConfig?.admissionControllerConfig?.enforceOnUpdates
                        }
                    />
                </div>
                <div className={wrapperMarginClassName}>
                    <div className={`pl-2 ${justifyBetweenClassName}`}>
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
                    <HelmValueWarning
                        currentValue={dynamicConfig.admissionControllerConfig.timeoutSeconds}
                        helmValue={
                            helmConfig?.dynamicConfig?.admissionControllerConfig?.timeoutSeconds
                        }
                    />
                </div>
                <div className={wrapperMarginClassName}>
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
                    <HelmValueWarning
                        currentValue={dynamicConfig.admissionControllerConfig.scanInline}
                        helmValue={helmConfig?.dynamicConfig?.admissionControllerConfig?.scanInline}
                    />
                </div>
                <div className={wrapperMarginClassName}>
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
                    <HelmValueWarning
                        currentValue={dynamicConfig.admissionControllerConfig.disableBypass}
                        helmValue={
                            helmConfig?.dynamicConfig?.admissionControllerConfig?.disableBypass
                        }
                    />
                </div>
            </div>
        </CollapsibleSection>
    );
};

export default DynamicConfigurationSection;
