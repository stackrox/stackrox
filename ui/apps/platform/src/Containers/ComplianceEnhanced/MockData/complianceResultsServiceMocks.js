export function mockGetComplianceScanResultsOverview() {
    return [
        {
            scanStats: {
                scanName: 'HIPPA Scan',
                checkStats: [
                    { count: 20, status: 'PASS' },
                    { count: 34, status: 'FAIL' },
                ],
                lastScan: '2023-08-03T04:49:58Z',
            },
            profileName: ['HIPPA'],
            cluster: [
                {
                    clusterId: '123',
                    clusterName: 'Primary',
                },
                {
                    clusterId: '234',
                    clusterName: 'Secondary',
                },
            ],
        },
        {
            scanStats: {
                scanName: 'CIS Scans',
                checkStats: [
                    { count: 20, status: 'PASS' },
                    { count: 34, status: 'FAIL' },
                ],
                lastScan: '2023-08-03T04:49:58Z',
            },
            profileName: ['profile-1', 'profile-2'],
            cluster: [
                {
                    clusterId: '123',
                    clusterName: 'Primary',
                },
            ],
        },
        {
            scanStats: {
                scanName: 'PCI Scans',
                checkStats: [
                    { count: 20, status: 'PASS' },
                    { count: 34, status: 'FAIL' },
                ],
                lastScan: '2023-08-03T04:49:58Z',
            },
            profileName: ['PCI'],
            cluster: [
                {
                    clusterId: '234',
                    clusterName: 'Secondary',
                },
            ],
        },
    ];
}
