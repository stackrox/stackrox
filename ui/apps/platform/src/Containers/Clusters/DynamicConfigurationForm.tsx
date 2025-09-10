import React from 'react';
import {
    Alert,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import SelectSingle from 'Components/SelectSingle';
import useMetadata from 'hooks/useMetadata';
import type { ClusterType, CompleteClusterConfig, DynamicClusterConfig } from 'types/cluster.proto';
import { getVersionedDocs } from 'utils/versioning';

import HelmValueWarning from './Components/HelmValueWarning';

export type DynamicConfigurationFormProps = {
    clusterType: ClusterType;
    dynamicConfig: DynamicClusterConfig;
    handleChange: (path: string, value: boolean | string) => void;
    handleChangeAdmissionControllerEnforcementBehavior: (value: boolean) => void;
    helmConfig: CompleteClusterConfig | null;
    isManagerTypeNonConfigurable: boolean;
};

function DynamicConfigurationForm({
    clusterType,
    dynamicConfig,
    handleChange,
    handleChangeAdmissionControllerEnforcementBehavior,
    helmConfig,
    isManagerTypeNonConfigurable,
}: DynamicConfigurationFormProps) {
    const { version } = useMetadata();

    const isLoggingSupported = clusterType === 'OPENSHIFT4_CLUSTER';

    // Assumptions:
    // TextInput element: property path is same as first argument of handleChange.
    // SelectSingle element: property path is same as value of id prop (which determines argument).
    // HelmValueWarning precedes FormHelperText element.
    return (
        <Form isWidthLimited>
            <FormGroup label="Custom default image registry">
                <TextInput
                    type="text"
                    value={dynamicConfig.registryOverride}
                    onChange={(_event, value) =>
                        handleChange('dynamicConfig.registryOverride', value)
                    }
                    isDisabled={isManagerTypeNonConfigurable}
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Set a value if the default registry is not docker.io in this cluster
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FormGroup label="Admission controller enforcement behavior">
                <SelectSingle
                    id="dynamicConfig.admissionControllerConfig.enabled"
                    value={
                        dynamicConfig.admissionControllerConfig.enabled ||
                        dynamicConfig.admissionControllerConfig.enforceOnUpdates
                            ? 'enabled'
                            : 'disabled'
                    }
                    handleSelect={(id, value) =>
                        handleChangeAdmissionControllerEnforcementBehavior(value === 'enabled')
                    }
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    <SelectOption value="enabled">Enforce policies</SelectOption>
                    <SelectOption value="disabled">No enforcement</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={
                        dynamicConfig.admissionControllerConfig.enabled ||
                        dynamicConfig.admissionControllerConfig.enforceOnUpdates
                    }
                    helmValue={
                        helmConfig?.dynamicConfig?.admissionControllerConfig?.enabled ||
                        helmConfig?.dynamicConfig?.admissionControllerConfig?.enforceOnUpdates
                    }
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Controls the policy enforcement configuration of the admission
                            controller. It determines whether the admission controller actively
                            blocks workloads and operations if they violate policies.
                        </HelperTextItem>
                        <HelperTextItem>
                            For more information, see{' '}
                            <ExternalLink>
                                <a
                                    href={getVersionedDocs(
                                        version,
                                        'operating/use-admission-controller-enforcement'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    RHACS documentation
                                </a>
                            </ExternalLink>
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FormGroup label="Admission controller bypass annotation">
                <SelectSingle
                    id="dynamicConfig.admissionControllerConfig.disableBypass"
                    value={
                        dynamicConfig.admissionControllerConfig.disableBypass
                            ? 'disabled'
                            : 'enabled'
                    }
                    handleSelect={(id, value) => handleChange(id, value === 'disabled')}
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    <SelectOption value="enabled">Enabled</SelectOption>
                    <SelectOption value="disabled">Disabled</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={dynamicConfig.admissionControllerConfig.disableBypass}
                    helmValue={helmConfig?.dynamicConfig?.admissionControllerConfig?.disableBypass}
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Allows teams to bypass admission controller in a monitored manner in the
                            event of an emergency
                        </HelperTextItem>
                        <HelperTextItem>
                            For more information, see{' '}
                            <ExternalLink>
                                <a
                                    href={getVersionedDocs(
                                        version,
                                        'operating/use-admission-controller-enforcement'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    RHACS documentation
                                </a>
                            </ExternalLink>
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FormGroup label="Cluster audit logging">
                <SelectSingle
                    id="tolerationsConfig.disabled"
                    value={dynamicConfig.disableAuditLogs ? 'disabled' : 'enabled'}
                    handleSelect={(id, value) => handleChange(id, value === 'disabled')}
                    isDisabled={isManagerTypeNonConfigurable || !isLoggingSupported}
                >
                    <SelectOption value="enabled">Enabled</SelectOption>
                    <SelectOption value="disabled">Disabled</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={dynamicConfig.disableAuditLogs}
                    helmValue={helmConfig?.dynamicConfig?.disableAuditLogs}
                />
                {!isLoggingSupported && (
                    <Alert
                        variant="warning"
                        title="Kubernetes and Openshift compatibility"
                        component="p"
                        isInline
                    >
                        This setting will not work for Kubernetes or OpenShift 3.x. To enable
                        logging, you must upgrade your cluster to OpenShift 4 or higher.
                    </Alert>
                )}
            </FormGroup>
        </Form>
    );
}

export default DynamicConfigurationForm;
