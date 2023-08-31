export type PassingRateData = {
    name: string;
    passing: number;
    link: string;
};

interface ComplianceScanStatsShim {
    scanName: string;
    numberOfChecks: number;
    numberOfFailingChecks: number;
    numberOfPassingChecks: number;
    lastScan: string;
}

export interface ComplianceScanResultsOverview {
    scanStats: ComplianceScanStatsShim;
    profileName: string[];
    clusterId: string[];
}

export interface ListComplianceScanResultsOverviewResponse {
    scanOverviews: ComplianceScanResultsOverview[];
}
