import type { InternalAxiosRequestConfig } from 'axios';

import consoleFetchAxiosAdapter from './consoleFetchAxiosAdapter';
import { ALL_NAMESPACES_KEY } from './constants';

const mockConsoleFetch = vi.hoisted(() => vi.fn());
vi.mock('@openshift-console/dynamic-plugin-sdk', () => ({
    consoleFetch: mockConsoleFetch,
}));

describe('consoleFetchAxiosAdapter', () => {
    describe('auth scope header injection', () => {
        const baseUrl = 'http://base';
        const metadataConfig = {
            url: '/v1/metadata',
            headers: {},
        } as InternalAxiosRequestConfig;
        const graphqlConfig = {
            url: '/api/graphql',
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

        it('should add namespace scope header when namespace is set for graphql requests', async () => {
            await consoleFetchAxiosAdapter(baseUrl, graphqlConfig, () => ({
                namespace: 'test-namespace',
            }));
            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'ACS-AUTH-NAMESPACE-SCOPE': 'test-namespace',
                    }),
                })
            );
        });

        it('should add wildcard scope header when namespace is set to all namespaces for graphql requests', async () => {
            await consoleFetchAxiosAdapter(baseUrl, graphqlConfig, () => ({
                namespace: ALL_NAMESPACES_KEY,
            }));
            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'ACS-AUTH-NAMESPACE-SCOPE': '*',
                    }),
                })
            );
        });

        it('should not add namespace scope header when namespace is set for non-graphql requests', async () => {
            await consoleFetchAxiosAdapter(baseUrl, metadataConfig, () => ({
                namespace: 'test-namespace',
            }));
            expect(mockConsoleFetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.not.objectContaining({
                        'ACS-AUTH-NAMESPACE-SCOPE': expect.anything(),
                    }),
                })
            );
        });

        it('should not add scope headers when scope is empty', async () => {
            const getScope = () => ({});

            await consoleFetchAxiosAdapter(baseUrl, metadataConfig, getScope);

            const { headers } = mockConsoleFetch.mock.calls[0][1];
            expect(headers['ACS-AUTH-NAMESPACE-SCOPE']).toBeUndefined();
        });

        it('should use default scope getter when none provided', async () => {
            await consoleFetchAxiosAdapter(baseUrl, metadataConfig);

            const { headers } = mockConsoleFetch.mock.calls[0][1];
            expect(headers['ACS-AUTH-NAMESPACE-SCOPE']).toBeUndefined();
        });

        it('should preserve existing headers', async () => {
            const configWithHeaders = {
                ...graphqlConfig,
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
                ...graphqlConfig,
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
