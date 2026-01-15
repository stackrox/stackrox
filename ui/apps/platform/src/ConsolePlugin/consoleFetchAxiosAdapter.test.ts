import type { InternalAxiosRequestConfig } from 'axios';

import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';

const mockConsoleFetch = vi.hoisted(() => vi.fn());
vi.mock('@openshift-console/dynamic-plugin-sdk', () => ({
    consoleFetch: mockConsoleFetch,
}));

describe('consoleFetchAxiosAdapter', () => {
    describe('auth scope header injection', () => {
        const baseUrl = 'http://base';
        const defaultConfig = {
            url: '/api/test',
            headers: {},
        } as InternalAxiosRequestConfig;

        beforeEach(() => {
            vi.clearAllMocks();
            mockConsoleFetch.mockResolvedValue({
                text: async () => '{"data":"test"}',
                headers: new Map(),
                statusText: 'OK',
                status: 200,
            });
        });

        it('should add namespace scope header when namespace is set', async () => {
            const getScope = () => ({ namespace: 'test-namespace' });

            await consoleFetchAxiosAdapter(baseUrl, defaultConfig, getScope);

            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'ACS-AUTH-NAMESPACE-SCOPE': 'test-namespace',
                    }),
                })
            );
        });

        it('should not add scope headers when scope is empty', async () => {
            const getScope = () => ({});

            await consoleFetchAxiosAdapter(baseUrl, defaultConfig, getScope);

            const { headers } = mockConsoleFetch.mock.calls[0][1];
            expect(headers['ACS-AUTH-NAMESPACE-SCOPE']).toBeUndefined();
        });

        it('should use default scope getter when none provided', async () => {
            await consoleFetchAxiosAdapter(baseUrl, defaultConfig);

            const { headers } = mockConsoleFetch.mock.calls[0][1];
            expect(headers['ACS-AUTH-NAMESPACE-SCOPE']).toBeUndefined();
        });

        it('should preserve existing headers', async () => {
            const configWithHeaders = {
                url: '/api/test',
                headers: {
                    'Content-Type': 'application/json',
                    'Custom-Header': 'custom-value',
                },
            } as unknown as InternalAxiosRequestConfig;

            await consoleFetchAxiosAdapter(baseUrl, configWithHeaders);

            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'Content-Type': 'application/json',
                        'Custom-Header': 'custom-value',
                    }),
                })
            );
        });

        it('should add scope headers alongside existing headers', async () => {
            const configWithHeaders = {
                url: '/api/test',
                headers: {
                    'Content-Type': 'application/json',
                },
            } as unknown as InternalAxiosRequestConfig;
            const getScope = () => ({ namespace: 'test-ns' });

            await consoleFetchAxiosAdapter(baseUrl, configWithHeaders, getScope);

            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'Content-Type': 'application/json',
                        'ACS-AUTH-NAMESPACE-SCOPE': 'test-ns',
                    }),
                })
            );
        });
    });
});
