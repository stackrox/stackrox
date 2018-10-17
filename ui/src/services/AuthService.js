import store from 'store';
import axios from 'axios';

const authProvidersUrl = '/v1/authProviders';
const accessTokenKey = 'access_token';
const requestedLocationKey = 'requested_location';

/**
 * Authentication HTTP Error that encapsulates HTTP errors related to user authentication and authorization.
 *
 * @class AuthHttpError
 * @extends {Error}
 */
export class AuthHttpError extends Error {
    constructor(message, code, cause) {
        super(message);
        this.name = 'AuthHttpError';
        this.code = code;
        this.cause = cause;
    }

    isAccessDenied = () => this.code === 403;

    isInvalidAuth = () => this.code === 401;
}

/**
 * Fetches authentication providers.
 *
 * @returns {Promise<Object, Error>} object with response property being an array of auth providers
 */
export function fetchAuthProviders() {
    return axios.get(`${authProvidersUrl}`).then(response => ({
        response: response.data.authProviders
    }));
}

/**
 * Saves auth provider either by creating a new one (in case ID is missed) or by updating existing one by ID.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete
 */
export function saveAuthProvider(authProvider) {
    return authProvider.id
        ? axios.patch(`${authProvidersUrl}/${authProvider.id}`, {
              name: authProvider.name,
              enabled: authProvider.enabled
          })
        : axios.post(authProvidersUrl, authProvider);
}

/**
 * Deletes auth provider by its ID.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete
 */
export function deleteAuthProvider(authProviderId) {
    if (!authProviderId) throw new Error('Auth provider ID must be defined');
    return axios.delete(`${authProvidersUrl}/${authProviderId}`);
}

/**
 * Deletes auth providers by a list of IDs.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete
 */
export function deleteAuthProviders(authProviderIds) {
    return Promise.all(authProviderIds.map(id => deleteAuthProvider(id)));
}

/**
 * Exchanges an external auth token for a Rox auth token.
 *
 * @param token the external auth token
 * @param type the type of authentication provider
 * @param state the state parameter
 * @param extra additional parameters
 * @returns {Promise} promise which is fulfilled when the request is complete
 */
export function exchangeAuthToken(token, type, state) {
    const data = {
        external_token: token,
        type,
        state
    };
    return axios.post(`${authProvidersUrl}/exchangeToken`, data).then(response => response.data);
}

/**
 * Calls the server to check auth status, rejects with error if auth status isn't valid.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchAuthStatus() {
    return axios.get('/v1/auth/status');
}

const getAccessToken = () => store.get(accessTokenKey) || null;
export const storeAccessToken = token => store.set(accessTokenKey, token);
export const clearAccessToken = () => store.remove(accessTokenKey);
export const isTokenPresent = () => !!getAccessToken();

export const storeRequestedLocation = location => store.set(requestedLocationKey, location);
export const getAndClearRequestedLocation = () => {
    const location = store.get(requestedLocationKey);
    store.remove(requestedLocationKey);
    return location;
};

const BEARER_TOKEN_PREFIX = `Bearer `;

function setAuthHeader(config, token) {
    const {
        headers: { Authorization, ...notAuthHeaders }
    } = config;
    // make sure new config doesn't have unnecessary auth header
    const newConfig = {
        ...config,
        headers: {
            ...notAuthHeaders
        }
    };
    if (token) newConfig.headers.Authorization = `${BEARER_TOKEN_PREFIX}${token}`;

    return newConfig;
}

function parseAuthTokenFromHeaders(headers) {
    if (
        !headers ||
        typeof headers.Authorization !== 'string' ||
        !headers.Authorization.startsWith(BEARER_TOKEN_PREFIX)
    ) {
        return null;
    }
    return headers.Authorization.substring(BEARER_TOKEN_PREFIX.length);
}

function addRequestInterceptor() {
    axios.interceptors.request.use(
        config => setAuthHeader(config, getAccessToken()),
        error => Promise.reject(error)
    );
}

function addResponseInterceptor(authHttpErrorHandler) {
    axios.interceptors.response.use(
        response => response,
        error => {
            const {
                response: { status },
                config
            } = error;
            if (status === 401 || status === 403) {
                const requestToken = parseAuthTokenFromHeaders(config.headers);
                const currentToken = getAccessToken();
                if (currentToken !== requestToken) {
                    // backend auth was enabled, but the request was made with old / empty token (e.g. multiple browser tabs open)
                    // in this case retry the request with a new token instead of failing
                    return axios.request(setAuthHeader(config, currentToken));
                }
                // we used the current / latest token and it failed
                const authError = new AuthHttpError('Authentication Error', status, error);
                if (authError.isInvalidAuth()) clearAccessToken(); // clear token since it's not valid
                authHttpErrorHandler(authError);
            }
            return Promise.reject(error);
        }
    );
}

let interceptorsAdded = false;

/**
 * Adds HTTP interceptors to pass authentication headers and catch auth/authz error responses.
 *
 * @param {!Function} authHttpErrorHandler handler that will be invoked with AuthHttpError
 */
export function addAuthInterceptors(authHttpErrorHandler) {
    if (interceptorsAdded) return;
    addRequestInterceptor();
    addResponseInterceptor(authHttpErrorHandler);
    interceptorsAdded = true;
}
