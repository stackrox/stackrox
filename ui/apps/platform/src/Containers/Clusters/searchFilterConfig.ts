import {
    clusterIdAttribute,
    clusterKubernetesVersionAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
    clusterPlatformTypeAttribute,
    clusterTypeAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import type {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterEntity,
    SelectSearchFilterOptions,
} from 'Components/CompoundSearchFilter/types';

export const statusSelectOptions: SelectSearchFilterOptions['options'] = [
    { label: 'Healthy', value: 'HEALTHY' },
    { label: 'Degraded', value: 'DEGRADED' },
    { label: 'Unhealthy', value: 'UNHEALTHY' },
    { label: 'Unavailable', value: 'UNAVAILABLE' },
    { label: 'Uninitialized', value: 'UNINITIALIZED' },
];

function createStatusAttribute(entity: string): CompoundSearchFilterAttribute {
    return {
        displayName: 'Status',
        filterChipLabel: `${entity} status`,
        searchTerm: `${entity} status`,
        inputType: 'select',
        inputProps: { options: statusSelectOptions },
    };
}

const admissionControlStatusAttribute = createStatusAttribute('Admission control');
const clusterStatusAttribute = createStatusAttribute('Cluster');
const collectorStatusAttribute = createStatusAttribute('Collector');
const scannerStatusAttribute = createStatusAttribute('Scanner');
const sensorStatusAttribute = createStatusAttribute('Sensor');

const lastContactAttributes: CompoundSearchFilterAttribute = {
    displayName: 'Date',
    filterChipLabel: 'Last contact',
    searchTerm: 'Last contact',
    inputType: 'date-picker',
};

const admissionControlSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Admission control',
    searchCategory: 'CLUSTERS',
    attributes: [admissionControlStatusAttribute],
};

const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [
        clusterIdAttribute,
        clusterNameAttribute,
        clusterLabelAttribute,
        clusterStatusAttribute,
        clusterTypeAttribute,
        clusterPlatformTypeAttribute,
        clusterKubernetesVersionAttribute,
    ],
};

const collectorSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Collector',
    searchCategory: 'CLUSTERS',
    attributes: [collectorStatusAttribute],
};

const lastContactSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Last contact',
    searchCategory: 'CLUSTERS',
    attributes: [lastContactAttributes],
};

const scannerSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Scanner',
    searchCategory: 'CLUSTERS',
    attributes: [scannerStatusAttribute],
};

const sensorSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Sensor',
    searchCategory: 'CLUSTERS',
    attributes: [sensorStatusAttribute],
};

export const searchFilterConfig = [
    clusterSearchFilterConfig,
    lastContactSearchFilterConfig,
    admissionControlSearchFilterConfig,
    collectorSearchFilterConfig,
    scannerSearchFilterConfig,
    sensorSearchFilterConfig,
];
