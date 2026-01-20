import { AnalyticsBrowser } from '@segment/analytics-next';
import Raven from 'raven-js';

export type AnalyticsSource = 'standalone' | 'console-plugin' | 'unknown';

// Module-level isolated analytics instance
// In Cypress tests, check for test mock instance first
// @ts-expect-error - Cypress test mock is not typed
let analyticsInstance: AnalyticsBrowser | null = window?.CYPRESS_ANALYTICS_INSTANCE ?? null;
let analyticsSource: AnalyticsSource = 'unknown'; // Default to unknown, must be set prior to initialization

/**
 * Sets the module-level analytics source for the application
 * Must be called once at the root of the application before initializing analytics
 *
 * @param source - The source of the analytics (standalone or console-plugin)
 */
export function setAnalyticsSource(source: Exclude<AnalyticsSource, 'unknown'>): void {
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
    if (analyticsInstance) {
        return;
    }

    if (analyticsSource === 'unknown') {
        Raven.captureMessage(
            'Analytics source is unknown, events will still be collected but will not be segmented by source.'
        );
        // eslint-disable-next-line no-console
        console.error(
            'Analytics is being initialized with an unknown source, this is almost certainly a developer error.'
        );
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

    analyticsInstance
        .addSourceMiddleware(({ payload, next }) => {
            const eventType = payload.type();

            // Source is added as a property (not context) so it's available for segmentation in Amplitude
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
