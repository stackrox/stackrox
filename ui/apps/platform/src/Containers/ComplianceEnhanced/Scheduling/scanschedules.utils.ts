import { ScanSchedule } from 'services/ComplianceEnhancedService';

export const initialScanSchedule: ScanSchedule = {
    scanName: '',
    clusters: [],
    scanConfig: {
        profiles: [],
        oneTimeScan: true,
        scanSchedule: null,
    },
};
