import React from 'react';

import CollapsibleSection from 'Components/CollapsibleSection';
import ToggleSwitch from 'Components/ToggleSwitch';

import {
    labelClassName,
    sublabelClassName,
    wrapperMarginClassName,
    inputTextClassName,
    divToggleOuterClassName,
    justifyBetweenClassName,
    inputNumberClassName,
} from 'constants/form.constants';
import { clusterTypes } from './cluster.helpers';
import HelmValueWarning from './Components/HelmValueWarning';

const DynamicConfigurationSection = ({
    handleChange,
    dynamicConfig,
    helmConfig,
    clusterType,
    isManagerTypeNonConfigurable,
}) => {
    const { registryOverride, admissionControllerConfig } = dynamicConfig;

    const isLoggingSupported = clusterType === clusterTypes.OPENSHIFT_4;

    // @TODO, replace open prop with dynamic logic, based on clusterType
    return (
        <CollapsibleSection title="Dynamic Configuration (syncs with Sensor)">
            <div className="bg-base-100 pb-3 pt-1 px-3 rounded shadow">
                <div className={wrapperMarginClassName}>
                    <label htmlFor="dynamicConfig.registryOverride" className={labelClassName}>
                        <span>Custom default image registry</span>
                        <br />
                        <span className={sublabelClassName}>
                            Set a value if the default registry is not docker.io in this cluster
                        </span>
                    </label>
                    <div className="flex" data-testid="input-wrapper">
                        <input
                            id="dynamicConfig.registryOverride"
                            name="dynamicConfig.registryOverride"
                            onChange={handleChange}
                            value={registryOverride}
                            className={inputTextClassName}
                            placeholder="image-mirror.example.com"
                            disabled={isManagerTypeNonConfigurable}
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
                            Enforce on Object Creates
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.admissionControllerConfig.enabled"
                            name="dynamicConfig.admissionControllerConfig.enabled"
                            toggleHandler={handleChange}
                            enabled={admissionControllerConfig.enabled}
                            disabled={isManagerTypeNonConfigurable}
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
                            Enforce on Object Updates
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                            name="dynamicConfig.admissionControllerConfig.enforceOnUpdates"
                            toggleHandler={handleChange}
                            enabled={admissionControllerConfig.enforceOnUpdates}
                            disabled={isManagerTypeNonConfigurable}
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
                            htmlFor="dynamicConfig.admissionControllerConfig.timeoutSeconds"
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
                            disabled={isManagerTypeNonConfigurable}
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
                            disabled={isManagerTypeNonConfigurable}
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
                            disabled={isManagerTypeNonConfigurable}
                        />
                    </div>
                    <HelmValueWarning
                        currentValue={dynamicConfig.admissionControllerConfig.disableBypass}
                        helmValue={
                            helmConfig?.dynamicConfig?.admissionControllerConfig?.disableBypass
                        }
                    />
                </div>
                <div className={wrapperMarginClassName}>
                    <div className={`${divToggleOuterClassName} ${justifyBetweenClassName}`}>
                        <label htmlFor="dynamicConfig.disableAuditLogs" className={labelClassName}>
                            Enable Cluster Audit Logging
                        </label>
                        <ToggleSwitch
                            id="dynamicConfig.disableAuditLogs"
                            name="dynamicConfig.disableAuditLogs"
                            disabled={!isLoggingSupported || isManagerTypeNonConfigurable}
                            toggleHandler={handleChange}
                            enabled={dynamicConfig.disableAuditLogs}
                            flipped
                        />
                    </div>
                    {!isLoggingSupported && (
                        <div className="border border-alert-200 bg-alert-200 p-2 rounded-b">
                            This setting will not work for Kubernetes or OpenShift 3.x. To enable
                            logging, you must upgrade your cluster to OpenShift 4 or higher.
                        </div>
                    )}
                    <HelmValueWarning
                        currentValue={dynamicConfig.disableAuditLogs}
                        helmValue={helmConfig?.dynamicConfig?.disableAuditLogs}
                    />
                </div>
            </div>
        </CollapsibleSection>
    );
};

export default DynamicConfigurationSection;
