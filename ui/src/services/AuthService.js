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
        ? axios.put(`${authProvidersUrl}/${authProvider.id}`, authProvider)
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
 * Calls the server to check auth status, rejects with error if auth status isn't valid.
 *
 * @returns {Promise<Object, Error>}
 */
export function fetchAuthStatus() {
    return axios.get('/v1/auth/status');
}

const getAccessToken = () => store.get(accessTokenKey);
export const storeAccessToken = token => store.set(accessTokenKey, token);
export const clearAccessToken = () => store.remove(accessTokenKey);
export const isTokenPresent = () => !!getAccessToken();

export const storeRequestedLocation = location => store.set(requestedLocationKey, location);
export const getAndClearRequestedLocation = () => {
    const location = store.get(requestedLocationKey);
    store.remove(requestedLocationKey);
    return location;
};

function addRequestInterceptor() {
    axios.interceptors.request.use(
        config => {
            const token = getAccessToken();
            if (!token) return config;
            // if there is a token available, then add it to auth header
            return {
                ...config,
                headers: {
                    ...config.headers,
                    Authorization: `Bearer ${token}`
                }
            };
        },
        error => Promise.reject(error)
    );
}

function addResponseInterceptor(authHttpErrorHandler) {
    axios.interceptors.response.use(
        response => response,
        error => {
            const { response: { status } } = error;
            if (status === 401 || status === 403) {
                const authError = new AuthHttpError('Authentication Error', status, error);
                if (authError.isInvalidAuth()) clearAccessToken(); // invalid auth means token isn't valid
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
