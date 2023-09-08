export function mockComplianceScanResultsOverview() {
    return {
        scanOverviews: [
            {
                scanStats: {
                    scanName: 'HIPPA Scan',
                    numberOfChecks: 54,
                    numberOfFailingChecks: 34,
                    numberOfPassingChecks: 20,
                    lastScan: '2023-08-03T04:49:58Z',
                },
                profileName: ['HIPPA'],
                clusterId: ['records-database'],
            },
            {
                scanStats: {
                    scanName: 'CIS Scans',
                    numberOfChecks: 78,
                    numberOfFailingChecks: 23,
                    numberOfPassingChecks: 55,
                    lastScan: '2023-08-03T04:49:58Z',
                },
                profileName: ['profile-1', 'profile-2'],
                clusterId: ['cluster-1', 'cluster-2', 'cluster-3'],
            },
            {
                scanStats: {
                    scanName: 'PCI Scans',
                    numberOfChecks: 45,
                    numberOfFailingChecks: 16,
                    numberOfPassingChecks: 29,
                    lastScan: '2023-08-03T04:49:58Z',
                },
                profileName: ['PCI'],
                clusterId: ['cluster-1', 'cluster-2'],
            },
        ],
    };
}
