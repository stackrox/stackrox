/* eslint-disable no-console */
/* eslint-disable no-multi-assign */
/* eslint-disable no-plusplus */
/* eslint-disable no-unused-expressions */
/* eslint-disable prefer-rest-params */

// the code below is generated from segment api with the exception of the writeKey/userId parameters as well as the analyticsIdentity call
// segment intends for the write key to be hardcoded but we are pulling it from the telemetry config service call and adding it using a parameter

export function initializeAnalytics(writeKey, proxyApiEndpoint, userId) {
    const analytics = (window.analytics = window.analytics || []);
    if (!analytics.initialize) {
        if (analytics.invoked) {
            window.console && console.error && console.error('Segment snippet included twice.');
        } else {
            analytics.invoked = !0;
            analytics.methods = [
                'trackSubmit',
                'trackClick',
                'trackLink',
                'trackForm',
                'pageview',
                'identify',
                'reset',
                'group',
                'track',
                'ready',
                'alias',
                'debug',
                'page',
                'once',
                'off',
                'on',
                'addSourceMiddleware',
                'addIntegrationMiddleware',
                'setAnonymousId',
                'addDestinationMiddleware',
            ];
            analytics.factory = function (e) {
                return function () {
                    const t = Array.prototype.slice.call(arguments);
                    t.unshift(e);
                    analytics.push(t);
                    return analytics;
                };
            };
            for (let e = 0; e < analytics.methods.length; e++) {
                const key = analytics.methods[e];
                analytics[key] = analytics.factory(key);
            }
            analytics.load = function (key, config) {
                const t = document.createElement('script');
                t.type = 'text/javascript';
                t.async = !0;

                const cdnBaseUrl = config?.cdnUrl || 'cdn.segment.com';
                t.src = `https://${cdnBaseUrl}/analytics.js/v1/${key}/analytics.min.js`;

                const n = document.getElementsByTagName('script')[0];
                n.parentNode.insertBefore(t, n);
                analytics._loadOptions = config;
            };
            analytics._writeKey = writeKey;
            analytics.SNIPPET_VERSION = '4.15.3';

            let analyticsLoadConfig;
            if (proxyApiEndpoint) {
                const proxyApiBaseUrl = proxyApiEndpoint.replace(/^https?:\/\//, '');
                analyticsLoadConfig = {
                    integrations: {
                        'Segment.io': {
                            apiHost: `${proxyApiBaseUrl}/v1`,
                        },
                    },
                };

                if (proxyApiEndpoint.includes('console.redhat.com')) {
                    analyticsLoadConfig.cdnUrl = 'console.redhat.com/connections/cdn';
                }
            }

            analytics.load(writeKey, analyticsLoadConfig);
            analytics.page();
            analytics.identify(userId);
        }
    }
}
