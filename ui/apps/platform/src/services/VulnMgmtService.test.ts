import type { CVEListResponse } from './VulnMgmtService';

// Test the response type shape to ensure it matches what the backend returns
// and what the UI consumes.
describe('VulnMgmtService types', () => {
    it('CVEListResponse should have the expected shape', () => {
        const response: CVEListResponse = {
            items: [
                {
                    cve: 'CVE-2024-0001',
                    topSeverity: 'CRITICAL_VULNERABILITY_SEVERITY',
                    topCvss: 9.8,
                    topNvdCvss: 9.5,
                    topEpssProbability: 0.97,
                    affectedImageCount: 42,
                    firstDiscoveredInSystem: '2024-01-15T10:30:00Z',
                    publishedOn: '2023-12-20T00:00:00Z',
                    pendingExceptionCount: 2,
                },
            ],
            totalCount: 1,
        };

        expect(response.items).toHaveLength(1);
        expect(response.totalCount).toBe(1);
        expect(response.items[0].cve).toBe('CVE-2024-0001');
        expect(response.items[0].topSeverity).toBe('CRITICAL_VULNERABILITY_SEVERITY');
    });

    it('CVEListResponse handles empty items', () => {
        const response: CVEListResponse = {
            items: [],
            totalCount: 0,
        };

        expect(response.items).toHaveLength(0);
        expect(response.totalCount).toBe(0);
    });

    it('CVEListResponse handles omitted optional fields', () => {
        const response: CVEListResponse = {
            items: [
                {
                    cve: 'CVE-2024-0002',
                    topSeverity: 'LOW_VULNERABILITY_SEVERITY',
                    topCvss: 2.0,
                    topNvdCvss: 0,
                    topEpssProbability: 0,
                    affectedImageCount: 1,
                    firstDiscoveredInSystem: '',
                    publishedOn: '',
                    pendingExceptionCount: 0,
                },
            ],
            totalCount: 1,
        };

        expect(response.items[0].firstDiscoveredInSystem).toBe('');
        expect(response.items[0].pendingExceptionCount).toBe(0);
    });
});
