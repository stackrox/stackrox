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
import type { SensorVersionCompatibility } from 'types/cluster.proto';

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

const sensorCompatibilityStatusOptions: SelectSearchFilterOptions<SensorVersionCompatibility>['options'] =
    [
        {
            label: 'Incompatible (Behind)',
            value: 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_BEHIND',
        },
        { label: 'Compatible (Behind)', value: 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_BEHIND' },
        { label: 'Matched', value: 'SENSOR_VERSION_COMPATIBILITY_MATCHED' },
        { label: 'Compatible (Ahead)', value: 'SENSOR_VERSION_COMPATIBILITY_COMPATIBLE_AHEAD' },
        { label: 'Incompatible (Ahead)', value: 'SENSOR_VERSION_COMPATIBILITY_INCOMPATIBLE_AHEAD' },
        { label: 'Unknown', value: 'SENSOR_VERSION_COMPATIBILITY_UNKNOWN' },
    ];

const sensorCompatibilityStatusAttribute: CompoundSearchFilterAttribute = {
    displayName: 'Compatibility status',
    filterChipLabel: 'Sensor compatibility status',
    searchTerm: 'Sensor Version Compatibility',
    inputType: 'select',
    inputProps: { options: sensorCompatibilityStatusOptions },
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
    attributes: [sensorStatusAttribute, sensorCompatibilityStatusAttribute],
};

export const searchFilterConfig = [
    admissionControlSearchFilterConfig,
    clusterSearchFilterConfig,
    collectorSearchFilterConfig,
    lastContactSearchFilterConfig,
    scannerSearchFilterConfig,
    sensorSearchFilterConfig,
];
