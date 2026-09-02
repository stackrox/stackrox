import type { DeploymentRiskSummary } from 'services/DeploymentsService';

const mockFetch = vi.hoisted(() => vi.fn());
vi.mock('services/DeploymentsService', () => ({
    fetchDeploymentRiskSummary: mockFetch,
}));

// The cache keeps state at module scope, so load a fresh copy for each test to
// keep them isolated (rather than adding a test-only reset to the module).
async function loadCache() {
    vi.resetModules();
    return import('./riskSummaryCache');
}

describe('riskSummaryCache', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('fetches and returns the summary for a deployment', async () => {
        mockFetch.mockResolvedValue({ summary: 'risk briefing' });
        const { fetchCachedDeploymentRiskSummary } = await loadCache();

        await expect(fetchCachedDeploymentRiskSummary('dep-1')).resolves.toEqual({
            summary: 'risk briefing',
        });
        expect(mockFetch).toHaveBeenCalledExactlyOnceWith('dep-1');
    });

    it('caches the result and does not fetch again on a subsequent call', async () => {
        mockFetch.mockResolvedValue({ summary: 'cached' });
        const { fetchCachedDeploymentRiskSummary } = await loadCache();

        await fetchCachedDeploymentRiskSummary('dep-1');
        await fetchCachedDeploymentRiskSummary('dep-1');

        expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('dedupes concurrent in-flight requests for the same deployment', async () => {
        let resolveFetch: (value: DeploymentRiskSummary) => void = () => {};
        mockFetch.mockImplementation(
            () =>
                new Promise<DeploymentRiskSummary>((resolve) => {
                    resolveFetch = resolve;
                })
        );
        const { fetchCachedDeploymentRiskSummary } = await loadCache();

        const first = fetchCachedDeploymentRiskSummary('dep-1');
        const second = fetchCachedDeploymentRiskSummary('dep-1');
        expect(mockFetch).toHaveBeenCalledTimes(1);

        resolveFetch({ summary: 'shared' });
        await expect(first).resolves.toEqual({ summary: 'shared' });
        await expect(second).resolves.toEqual({ summary: 'shared' });
    });

    it('caches per deployment id', async () => {
        mockFetch.mockImplementation((id: string) => Promise.resolve({ summary: `summary ${id}` }));
        const { fetchCachedDeploymentRiskSummary, peekCachedDeploymentRiskSummary } =
            await loadCache();

        await fetchCachedDeploymentRiskSummary('dep-1');
        await fetchCachedDeploymentRiskSummary('dep-2');

        expect(mockFetch).toHaveBeenCalledTimes(2);
        expect(peekCachedDeploymentRiskSummary('dep-1')).toEqual({ summary: 'summary dep-1' });
        expect(peekCachedDeploymentRiskSummary('dep-2')).toEqual({ summary: 'summary dep-2' });
    });

    it('does not cache failures and retries on the next attempt', async () => {
        mockFetch
            .mockRejectedValueOnce(new Error('boom'))
            .mockResolvedValueOnce({ summary: 'recovered' });
        const { fetchCachedDeploymentRiskSummary, peekCachedDeploymentRiskSummary } =
            await loadCache();

        await expect(fetchCachedDeploymentRiskSummary('dep-1')).rejects.toThrow('boom');
        expect(peekCachedDeploymentRiskSummary('dep-1')).toBeUndefined();

        await expect(fetchCachedDeploymentRiskSummary('dep-1')).resolves.toEqual({
            summary: 'recovered',
        });
        expect(mockFetch).toHaveBeenCalledTimes(2);
        expect(peekCachedDeploymentRiskSummary('dep-1')).toEqual({ summary: 'recovered' });
    });

    it('peek returns undefined before a summary has been fetched', async () => {
        const { peekCachedDeploymentRiskSummary } = await loadCache();
        expect(peekCachedDeploymentRiskSummary('dep-1')).toBeUndefined();
    });
});
