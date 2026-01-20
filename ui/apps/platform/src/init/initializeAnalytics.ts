import { AnalyticsBrowser } from '@segment/analytics-next';
import Raven from 'raven-js';

// Module-level isolated analytics instance
// In Cypress tests, check for test mock instance first
// @ts-expect-error - Cypress test mock is not typed
let analyticsInstance: AnalyticsBrowser | null = window?.CYPRESS_ANALYTICS_INSTANCE ?? null;

/**
 * Initializes Segment analytics once, encapsulate in module scope
 *
 * @param writeKey - The Segment write key for the source
 * @param proxyApiEndpoint - Optional proxy endpoint for API calls
 * @param userId - Optional user ID to identify the user
 */
export function initializeAnalytics(
    writeKey: string,
    proxyApiEndpoint?: string,
    userId?: string
): void {
    // Prevent duplicate initialization
    if (analyticsInstance) {
        return;
    }

    let cdnURL: string | undefined;
    let apiHost: string | undefined;

    if (proxyApiEndpoint) {
        const proxyApiBaseUrl = proxyApiEndpoint.replace(/^https?:\/\//, '');
        apiHost = `${proxyApiBaseUrl}/v1`;

        if (proxyApiEndpoint.includes('console.redhat.com')) {
            cdnURL = 'https://console.redhat.com/connections/cdn';
        }
    }

    analyticsInstance = AnalyticsBrowser.load(
        { writeKey, ...(cdnURL ? { cdnURL } : {}) },
        {
            integrations: {
                'Segment.io': {
                    ...(apiHost ? { apiHost } : {}),
                },
            },
        }
    );

    if (userId) {
        analyticsInstance.identify(userId).catch((error) => {
            Raven.captureException(error);
        });
    }
}

/**
 * Returns the initialized analytics instance
 * @returns The analytics instance, or null if not initialized
 */
export function getAnalytics(): AnalyticsBrowser | null {
    return analyticsInstance;
}
