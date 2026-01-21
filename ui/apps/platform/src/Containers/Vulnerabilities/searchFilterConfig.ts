/*
 * Search filter configurations for vulnerability views.
 *
 * If you add a new filter config that should be available in view-based reports,
 * add it to the viewBasedReportSearchFilterConfigs array at the bottom of this file.
 */

import type {
    CompoundSearchFilterEntity,
    SelectExclusiveSingleSearchFilterAttribute,
    SelectSearchFilterAttribute,
    SelectSearchFilterOption,
} from 'Components/CompoundSearchFilter/types';
import {
    clusterIdAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
    clusterPlatformTypeAttribute,
    clusterTypeAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import { Annotation, ID, Label, Name } from 'Components/CompoundSearchFilter/attributes/deployment';
import { imageAttributes } from 'Components/CompoundSearchFilter/attributes/image';
import { imageCVEAttributes } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { imageComponentAttributes } from 'Components/CompoundSearchFilter/attributes/imageComponent';
import { namespaceAttributes } from 'Components/CompoundSearchFilter/attributes/namespace';
import { nodeAttributes } from 'Components/CompoundSearchFilter/attributes/node';
import { nodeCVEAttributes } from 'Components/CompoundSearchFilter/attributes/nodeCVE';
import { nodeComponentAttributes } from 'Components/CompoundSearchFilter/attributes/nodeComponent';
import { platformCVEAttributes } from 'Components/CompoundSearchFilter/attributes/platformCVE';
import {
    VirtualMachineCVEName,
    VirtualMachineComponentName,
    VirtualMachineComponentVersion,
    VirtualMachineID,
    VirtualMachineName,
} from 'Components/CompoundSearchFilter/attributes/virtualMachine';

export const nodeSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: nodeAttributes,
};

export const nodeCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'NODE_VULNERABILITIES',
    attributes: nodeCVEAttributes,
};

export const nodeComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: nodeComponentAttributes,
};

export const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

export const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES_V2', // flat CVE data model
    attributes: imageCVEAttributes,
};

export const imageComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS_V2', // flat CVE data model
    attributes: imageComponentAttributes,
};

export const deploymentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: [Annotation, ID, Label, Name],
};

export const namespaceSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: namespaceAttributes,
};

export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [
        clusterIdAttribute,
        clusterLabelAttribute,
        clusterNameAttribute,
        clusterPlatformTypeAttribute,
        clusterTypeAttribute,
    ],
};

export const platformCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES',
    attributes: platformCVEAttributes,
};

export const virtualMachinesSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Virtual machine',
    searchCategory: 'VIRTUAL_MACHINES',
    attributes: [VirtualMachineID, VirtualMachineName],
};

export const virtualMachineCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'SEARCH_UNSET', // doesn't matter since we don't have autocomplete for virtual machines
    attributes: [VirtualMachineCVEName],
};

export const virtualMachineComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Component',
    searchCategory: 'SEARCH_UNSET', // doesn't matter since we don't have autocomplete for virtual machines
    attributes: [VirtualMachineComponentName, VirtualMachineComponentVersion],
};

export const virtualMachinesClusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [clusterNameAttribute, clusterIdAttribute],
};

// attributes for separate search filter elements in AdvancedFiltersToolbar.tsx file

export const attributeForSnoozed: SelectExclusiveSingleSearchFilterAttribute = {
    displayName: 'CVE snoozed',
    filterChipLabel: 'CVE snoozed',
    searchTerm: 'CVE Snoozed',
    inputType: 'select-exclusive-single', // placeholder because interaction is Show snoozed CVEs button
    inputProps: {
        options: [
            { label: 'true', value: 'true' }, // Snoozed
            { label: 'false', value: 'false' }, // Observed
        ],
    },
};

const optionsForFixable: SelectSearchFilterOption[] = [
    { label: 'Fixable', value: 'Fixable' }, // true?
    { label: 'Not fixable', value: 'Not fixable' }, // false?
]; // fixableStatuses in types.ts file?

export const attributeForClusterCveFixable: SelectSearchFilterAttribute = {
    displayName: 'CVE status',
    filterChipLabel: 'CVE status',
    searchTerm: 'CLUSTER CVE FIXABLE', // why ALL CAPS instead of 'Cluster CVE Fixable'
    inputType: 'select',
    inputProps: {
        options: optionsForFixable,
    },
};

export const attributeForFixable: SelectSearchFilterAttribute = {
    displayName: 'CVE status',
    filterChipLabel: 'CVE status',
    searchTerm: 'FIXABLE', // why ALL CAPS instead of 'Fixable'
    inputType: 'select',
    inputProps: {
        options: optionsForFixable,
    },
};

export const attributeForSeverity: SelectSearchFilterAttribute = {
    displayName: 'CVE severity',
    filterChipLabel: 'CVE severity',
    searchTerm: 'SEVERITY', // why ALL CAPS instead of 'Severity'
    inputType: 'select',
    inputProps: {
        options: [
            { label: 'Critical', value: 'Critical' },
            { label: 'Important', value: 'Important' },
            { label: 'Moderate', value: 'Moderate' },
            { label: 'Low', value: 'Low' },
            { label: 'Unknown', value: 'Unknown' },
        ], // vulnerabilitySeverityLabels in types.ts file?
    },
};

// This array includes filter configs that are relevant to view-based reports.
// Only add configs here if they should be available as filters in vulnerability reports.
export const viewBasedReportSearchFilterConfigs = [
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
];
