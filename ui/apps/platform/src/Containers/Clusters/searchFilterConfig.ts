import { clusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import {
    CompoundSearchFilterAttribute,
    CompoundSearchFilterEntity,
} from 'Components/CompoundSearchFilter/types';

function createStatusAttributes(entity: string): CompoundSearchFilterAttribute {
    return {
        displayName: 'Status',
        filterChipLabel: `${entity} status`,
        searchTerm: `${entity} status`,
        inputType: 'select',
        inputProps: {
            options: [
                { label: 'Degraded', value: 'DEGRADED' },
                { label: 'Healthy', value: 'HEALTHY' },
                { label: 'Unavailable', value: 'UNAVAILABLE' },
                { label: 'Unhealthy', value: 'UNHEALTHY' },
                { label: 'Uninitialized', value: 'UNINITIALIZED' },
            ],
        },
    };
}

const admissionControlStatusAttribute = createStatusAttributes('Admission control');
const clusterStatusAttribute = createStatusAttributes('Cluster');
const collectorStatusAttribute = createStatusAttributes('Collector');
const scannerStatusAttribute = createStatusAttributes('Scanner');
const sensorStatusAttribute = createStatusAttributes('Sensor');

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
    attributes: [...clusterAttributes, clusterStatusAttribute],
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
