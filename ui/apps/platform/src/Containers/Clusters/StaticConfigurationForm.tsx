import React from 'react';
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
import type { Cluster } from 'types/cluster.proto';
import { getVersionedDocs } from 'utils/versioning';

import { clusterTypeOptions, runtimeOptions } from './cluster.helpers';
import HelmValueWarning from './Components/HelmValueWarning';

export type StaticConfigurationFormProps = {
    selectedCluster: Cluster;
    isManagerTypeNonConfigurable: boolean;
    handleChange: (path: string, value: boolean | string) => void;
};

function StaticConfigurationForm({
    selectedCluster,
    isManagerTypeNonConfigurable,
    handleChange,
}: StaticConfigurationFormProps) {
    const { version } = useMetadata();

    const filteredClusterTypeOptions = clusterTypeOptions.filter((option) => {
        // For (majority) view scenario: display any type.
        // For (minority) selection scenario: omit obsolete OpenShift 3 option.
        return isManagerTypeNonConfigurable || option.value !== 'OPENSHIFT_CLUSTER';
    });

    const filteredRuntimeOptions = runtimeOptions.filter((option) => {
        // EBPF has been removed for secured clusters >= 4.5, but
        // needs to be displayed for clusters on older versions.
        //
        // If the manager type is configurable (i.e. not helm or operator)
        // we don't want EBPF as a selectable option, so filter it out,
        // otherwise include all options so it is displayed correctly.
        return isManagerTypeNonConfigurable || option.value !== 'EBPF';
    });

    // Assumptions:
    // TextInput element: property path is same as first argument of handleChange.
    // SelectSingle element: property path is same as value of id prop (which determines argument).
    // HelmValueWarning precedes FormHelperText element.
    return (
        <Form isWidthLimited>
            <FormGroup label="Cluster name" isRequired>
                <TextInput
                    type="text"
                    value={selectedCluster.name}
                    onChange={(_event, value) => handleChange('name', value)}
                    isDisabled={Boolean(selectedCluster.id)}
                    isRequired
                />
            </FormGroup>
            <FormGroup label="Cluster type" isRequired>
                <SelectSingle
                    id="type"
                    value={selectedCluster.type}
                    handleSelect={handleChange}
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    {filteredClusterTypeOptions.map(({ label, value }) => (
                        <SelectOption key={value} value={value}>
                            {label}
                        </SelectOption>
                    ))}
                </SelectSingle>
                <HelmValueWarning
                    currentValue={selectedCluster.type}
                    helmValue={selectedCluster?.helmConfig?.staticConfig?.type}
                />
            </FormGroup>
            <FormGroup label="Main image repository" isRequired>
                <TextInput
                    type="text"
                    value={selectedCluster.mainImage}
                    onChange={(_event, value) => handleChange('mainImage', value)}
                    isDisabled={isManagerTypeNonConfigurable}
                    isRequired
                />
                <HelmValueWarning
                    currentValue={selectedCluster.mainImage}
                    helmValue={selectedCluster?.helmConfig?.staticConfig?.mainImage}
                />
            </FormGroup>
            <FormGroup label="Central API endpoint (include port)" isRequired>
                <TextInput
                    type="text"
                    value={selectedCluster.centralApiEndpoint}
                    onChange={(_event, value) => handleChange('centralApiEndpoint', value)}
                    isDisabled={isManagerTypeNonConfigurable}
                    isRequired
                />
                <HelmValueWarning
                    currentValue={selectedCluster.centralApiEndpoint}
                    helmValue={selectedCluster?.helmConfig?.staticConfig?.centralApiEndpoint}
                />
            </FormGroup>
            <FormGroup label="Collection method" isRequired>
                <SelectSingle
                    id="collectionMethod"
                    value={selectedCluster.collectionMethod}
                    handleSelect={handleChange}
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    {filteredRuntimeOptions.map(({ label, value }) => (
                        <SelectOption key={value} value={value}>
                            {label}
                        </SelectOption>
                    ))}
                </SelectSingle>
                <HelmValueWarning
                    currentValue={selectedCluster.collectionMethod}
                    helmValue={selectedCluster?.helmConfig?.staticConfig?.collectionMethod}
                />
            </FormGroup>
            <FormGroup label="Collector image repository (uses Main image repository by default)">
                <TextInput
                    type="text"
                    value={selectedCluster.collectorImage}
                    onChange={(_event, value) => handleChange('collectorImage', value)}
                    isDisabled={isManagerTypeNonConfigurable}
                />
                <HelmValueWarning
                    currentValue={selectedCluster.collectorImage}
                    helmValue={selectedCluster?.helmConfig?.staticConfig?.collectorImage}
                />
            </FormGroup>
            <FormGroup label="Admission controller failure policy" isRequired>
                <SelectSingle
                    id="admissionControllerFailOnError"
                    value={
                        selectedCluster.admissionControllerFailOnError ? 'failClosed' : 'failOpen'
                    }
                    handleSelect={(id, value) => handleChange(id, value === 'failClosed')}
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    <SelectOption value="failOpen">Fail open</SelectOption>
                    <SelectOption value="failClosed">Fail closed</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={selectedCluster.admissionControllerFailOnError}
                    helmValue={
                        selectedCluster?.helmConfig?.staticConfig?.admissionControllerFailOnError
                    }
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Defines how the admission controller reacts when an error or timeout
                            prevents policy evaluation.
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
            <FormGroup label="Taint tolerations" isRequired>
                <SelectSingle
                    id="tolerationsConfig.disabled"
                    value={selectedCluster.tolerationsConfig?.disabled ? 'disabled' : 'enabled'}
                    handleSelect={(id, value) => handleChange(id, value === 'disabled')}
                    isDisabled={isManagerTypeNonConfigurable}
                >
                    <SelectOption value="enabled">Enabled</SelectOption>
                    <SelectOption value="eisabled">Disabled</SelectOption>
                </SelectSingle>
                <HelmValueWarning
                    currentValue={selectedCluster?.tolerationsConfig?.disabled}
                    helmValue={
                        selectedCluster?.helmConfig?.staticConfig?.tolerationsConfig?.disabled
                    }
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>
                            Tolerate all taints to run on all nodes of this cluster
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
        </Form>
    );
}

export default StaticConfigurationForm;
