export function mockGetComplianceScanResultsOverview() {
    return [
        {
            scanStats: {
                id: '123456',
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
                id: '234567',
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
                id: '345678',
                scanName: 'PCI Scans',
                numberOfChecks: 45,
                numberOfFailingChecks: 16,
                numberOfPassingChecks: 29,
                lastScan: '2023-08-03T04:49:58Z',
            },
            profileName: ['PCI'],
            clusterId: ['cluster-1', 'cluster-2'],
        },
    ];
}
