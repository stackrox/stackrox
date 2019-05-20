import dateFns from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';

const enabledOrDisabled = value => (value ? 'Enabled' : 'Disabled');

const collectionMethodMap = {
    NO_COLLECTION: 'None',
    KERNEL_MODULE: 'Kernel Module',
    EBPF: 'eBPF'
};

export const formatCollectionMethod = cluster => collectionMethodMap[cluster.collectionMethod];
export const formatAdmissionController = cluster => enabledOrDisabled(cluster.admissionController);

export const checkInLabel = 'Last Check-In';
export const formatLastCheckIn = cluster => {
    if (cluster.status && cluster.status.lastContact) {
        return dateFns.format(cluster.status.lastContact, dateTimeFormat);
    }
    return 'N/A';
};

export const sensorVersionLabel = 'Current Sensor Version';
export const formatSensorVersion = cluster =>
    (cluster.status && cluster.status.sensorVersion) || 'Not Running';
