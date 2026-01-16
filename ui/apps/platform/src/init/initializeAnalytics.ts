import { AnalyticsBrowser } from '@segment/analytics-next';
import Raven from 'raven-js';

// Module-level isolated analytics instance
// In Cypress tests, check for test mock instance first
let analyticsInstance: AnalyticsBrowser | null =
    // @ts-expect-error - Cypress test mock is not typed
    (typeof window !== 'undefined' && window.CYPRESS_ANALYTICS_INSTANCE) ?? null;
let analyticsSource: AnalyticsSource = 'standalone'; // Default to standalone

export type AnalyticsSource = 'standalone' | 'console-plugin';

/**
 * Sets the module-level analytics source for the application
 * Should be called once at the root of the application
 *
 * @param source - The source of the analytics (standalone or console-plugin)
 */
export function setAnalyticsSource(source: AnalyticsSource): void {
    analyticsSource = source;
}

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
    if (analyticsInstance !== null) {
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
        { writeKey, ...(cdnURL && { cdnURL }) },
        {
            integrations: {
                'Segment.io': {
                    ...(apiHost && { apiHost }),
                },
            },
        }
    );

    analyticsInstance
        .addSourceMiddleware(({ payload, next }) => {
            // Source is added as a property (not context) so it's available for segmentation in Amplitude
            const eventType = payload.type();
            if (eventType === 'track' || eventType === 'page') {
                // eslint-disable-next-line no-param-reassign
                payload.obj.properties = {
                    ...payload.obj.properties,
                    acsApplicationSource: analyticsSource,
                };
            }
            next(payload);
        })
        .catch((error) => {
            Raven.captureException(error);
        });

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
