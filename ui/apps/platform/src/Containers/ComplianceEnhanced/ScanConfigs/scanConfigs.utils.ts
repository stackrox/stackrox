import { ScanConfig } from 'services/ComplianceEnhancedService';

export const initialScanConfig: ScanConfig = {
    scanName: '',
    clusters: [],
    scanConfig: {
        profiles: [],
        oneTimeScan: true,
        scanSchedule: null,
    },
};
