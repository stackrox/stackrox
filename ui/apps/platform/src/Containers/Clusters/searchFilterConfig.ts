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

const admissionControlStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Admission control status',
    searchTerm: 'Admission Control Status',
    inputType: 'select',
    inputProps: { options: statusSelectOptions },
};
const clusterStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Cluster status',
    searchTerm: 'Cluster Status',
    inputType: 'select',
    inputProps: { options: statusSelectOptions },
};

const collectorStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Collector status',
    searchTerm: 'Collector Status',
    inputType: 'select',
    inputProps: { options: statusSelectOptions },
};

const scannerStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Scanner status',
    searchTerm: 'Scanner Status',
    inputType: 'select',
    inputProps: { options: statusSelectOptions },
};

const sensorStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Sensor status',
    searchTerm: 'Sensor Status',
    inputType: 'select',
    inputProps: { options: statusSelectOptions },
};

const lastContactAttributes: CompoundSearchFilterAttribute = {
    displayName: 'Date',
    filterChipLabel: 'Last contact',
    searchTerm: 'Last Contact',
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
        clusterKubernetesVersionAttribute,
        clusterLabelAttribute,
        clusterNameAttribute,
        clusterPlatformTypeAttribute,
        clusterStatusAttribute,
        clusterTypeAttribute,
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
    admissionControlSearchFilterConfig,
    clusterSearchFilterConfig,
    collectorSearchFilterConfig,
    lastContactSearchFilterConfig,
    scannerSearchFilterConfig,
    sensorSearchFilterConfig,
];
