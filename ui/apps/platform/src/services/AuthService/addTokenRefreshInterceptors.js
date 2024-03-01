/**
 * This module provides handling of refreshing the access token when any request gets declined with "401 Unauthorized".
 * There are two major principles how the logic works:
 *  1. Whenever the refresh token operation is in progress:
 *   - stall all new requests until its finished (except those using `doNotStallRequestConfigMarker`);
 *   - for every failed request attempt to retry once token is refreshed.
 *  2. If a request fails with "401 Unauthorized":
 *   - if token refresh operation is in progress, retry after its done (see 1.);
 *   - if the request used one access token, yet we possess already another one, then retry with a new one;
 *   - if the request hasn't been just retried after token refresh attempt, then start refresh token operation;
 *   - otherwise, escalate to the proper handler as user intervention is required, and fail the request.
 *
 * Note that by default it's an aggressive behavior with stalling all requests during the refresh token operation, and
 * it can lead to undesired issues, like the request to refresh the token gets stalled and deadlock occurring.
 * Normally it wouldn't happen if refresh token operation upon invocation synchronously calls refresh token API (or uses
 * another Axios instance for it). Yet if call to refresh token API cannot be done synchronously, then the additional
 * config `doNotStallRequestConfigMarker` should be applied to the call to never be stalled by these interceptors.
 * Additionally, configuration option `doNotStallRequests` disables logic of stalling new requests entirely.
 *
 * Honorable mentioning: https://github.com/Flyrell/axios-auth-refresh is almost good, yet:
 *  - it does `axios.Cancel` for some reason (yeah, global axios)
 *  - it doesn't allow to bypass request stalling logic, causing deadlock mentioned above
 *  - it's not smart enough to retry requests in case refresh token has been updated while they were in progress
 * Maybe eventually this one can be merged with `axios-auth-refresh`.
 *
 * @module services/AuthService/addTokenRefreshInterceptors
 */

/**
 * This function is called to refresh the token.
 * @callback ExtractAccessTokenFunc
 * @param {Object} config - Request config
 * @returns {string} Token extracted from the request config if any
 */

/**
 * This function is called when cannot recover from request auth error w/o user intervention.
 * @callback AuthErrorHandler
 * @param {Object} error - Error object from Axios instance
 */

/**
 * Detaches previously attached interceptors.
 * @callback DetachInterceptorsFunc
 */

const doNotStallRequestConfigMarker = '@@refresh-token-interceptor/do-not-stall-request';
const retriedAfterTokenRefreshMarker = '@@refresh-token-interceptor/retried-after-token-refresh';

function retry(axios, config) {
    return axios.request({ ...config, [retriedAfterTokenRefreshMarker]: true });
}

/**
 * Config to be applied to the requests that should never be stalled during the
 * refresh token operation, i.e. they will proceed as usual.
 * @type {Object}
 */
export const doNotStallRequestConfig = {
    [doNotStallRequestConfigMarker]: true,
};

function addRequestInterceptor(axios, refreshTokenOpPromise) {
    return axios.interceptors.request.use((config) => {
        if (config[doNotStallRequestConfigMarker]) {
            return config;
        }

        // stall all other requests until token refresh operation is finished
        return refreshTokenOpPromise.then(() => ({
            ...config,
            [retriedAfterTokenRefreshMarker]: true,
        }));
    });
}

/**
 * Attaches interceptors to handle Axios request failures that fail with "401 Unauthorized", and will trigger
 * token refresh operation using the provided `accessTokenManager`. Note that not providing valid `extractAccessToken`
 * config option will result to handler to not attempt to retry the request that failed but used the token different
 * from `accessTokenManager.getToken()`.
 *
 * @param {!Object} axios - Axios instance to use and attach interceptors to
 * @param {!Object} accessTokenManager - Instance of `AccessTokenManager` class
 * @param {Object} [options] - Configuration options
 * @param {AuthErrorHandler} [options.handleAuthError] - Handler for auth access error user must resolve
 * @param {boolean} [options.doNotStallRequests] - Skips logic of stalling requests while token is being refreshed
 * @returns {DetachInterceptorsFunc} Function to call to detach interceptors
 */
export default function addTokenRefreshInterceptors(axios, accessTokenManager, options = {}) {
    const refreshTokenOpListener = (opPromise) => {
        const interceptor = addRequestInterceptor(axios, opPromise);
        // remove interceptor as soon as token is refreshed
        opPromise.then(() => {
            axios.interceptors.request.eject(interceptor);
        });
    };
    if (!options.doNotStallRequests) {
        accessTokenManager.onRefreshTokenStarted(refreshTokenOpListener);
    }

    const interceptor = axios.interceptors.response.use(
        (response) => response,
        (error) => {
            if (!error.response || error.response.status !== 401) {
                return Promise.reject(error);
            }
            const { config } = error;

            // if we're in the middle of refreshing the token, retry after it's done
            const refreshTokenOpPromise = accessTokenManager.getRefreshTokenOpPromise();
            if (refreshTokenOpPromise != null) {
                return refreshTokenOpPromise.then(() => retry(axios, config));
            }

            // If we are currently in the midst of receiving an auth response, then wait
            // until we received all related information and cookies and retry afterward.
            // This is an edge case and may only happen during login.
            const dispatchResponsePromise = accessTokenManager.getDispatchResponsePromise();
            if (dispatchResponsePromise != null) {
                return dispatchResponsePromise.then(() => retry(axios, config));
            }

            // our access token is no good, try to refresh unless this request failed just after token refresh
            if (!config[retriedAfterTokenRefreshMarker]) {
                return accessTokenManager.refreshToken().then(() => retry(axios, config));
            }

            // ran out of options to handle the error w/o user intervention
            if (options.handleAuthError) {
                options.handleAuthError(error);
            }
            return Promise.reject(error);
        }
    );

    return () => {
        if (!options.doNotStallRequests) {
            accessTokenManager.removeRefreshTokenListener(refreshTokenOpListener);
        }
        axios.interceptors.response.eject(interceptor);
    };
}
