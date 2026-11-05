import {
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
import type { CompleteClusterConfig, DynamicClusterConfig } from 'types/cluster.proto';
import { getVersionedDocs } from 'utils/versioning';

import HelmValueWarning from './Components/HelmValueWarning';

export type DynamicConfigurationFormProps = {
    dynamicConfig: DynamicClusterConfig;
    helmConfig: CompleteClusterConfig | null;
};

function DynamicConfigurationForm({ dynamicConfig, helmConfig }: DynamicConfigurationFormProps) {
    const { version } = useMetadata();

    // Assumptions:
    // TextInput element: property path is same as first argument of handleChange.
    // SelectSingle element: property path is same as value of id prop (which determines argument).
    // HelmValueWarning precedes FormHelperText element.
    return (
        <Form isWidthLimited>
            <FormGroup
                fieldId="dynamicConfig.registryOverride"
                label="Custom default image registry"
            >
                <TextInput
                    id="dynamicConfig.registryOverride"
                    type="text"
                    value={dynamicConfig.registryOverride}
                    isDisabled
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Set a value if the default registry is not docker.io in this cluster
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
                    handleSelect={() => {}}
                    isDisabled
                    isFullWidth={false}
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
                                        'operating/using-policies-to-provide-security#understand-admission-controller-enforcement_use-admission-controller-enforcement'
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
                    id="dynamicConfig.disableAuditLogs"
                    value={dynamicConfig.disableAuditLogs ? 'disabled' : 'enabled'}
                    handleSelect={() => {}}
                    isDisabled
                    isFullWidth={false}
                >
                    <SelectOption value="enabled">Enabled</SelectOption>
                    <SelectOption value="disabled">Disabled</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={dynamicConfig.disableAuditLogs}
                    helmValue={helmConfig?.dynamicConfig?.disableAuditLogs}
                />
            </FormGroup>
            <FormGroup label="Automatically lock process baselines">
                <SelectSingle
                    id="dynamicConfig.autoLockProcessBaselinesConfig.enabled"
                    value={
                        dynamicConfig.autoLockProcessBaselinesConfig?.enabled
                            ? 'enabled'
                            : 'disabled'
                    }
                    handleSelect={() => {}}
                    isDisabled
                    isFullWidth={false}
                >
                    <SelectOption value="enabled">Enabled</SelectOption>
                    <SelectOption value="disabled">Disabled</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={dynamicConfig.autoLockProcessBaselinesConfig?.enabled}
                    helmValue={helmConfig?.dynamicConfig?.autoLockProcessBaselinesConfig?.enabled}
                />
            </FormGroup>
        </Form>
    );
}

export default DynamicConfigurationForm;
