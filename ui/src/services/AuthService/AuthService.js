import store from 'store';

import axios from 'services/instance';
import queryString from 'qs';
import AccessTokenManager from './AccessTokenManager';
import addTokenRefreshInterceptors, {
    doNotStallRequestConfig,
} from './addTokenRefreshInterceptors';

const authProvidersUrl = '/v1/authProviders';
const authLoginProvidersUrl = '/v1/login/authproviders';
const tokenRefreshUrl = '/sso/session/tokenrefresh';
const logoutUrl = '/sso/session/logout';

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
    return axios.get(`${authProvidersUrl}`).then((response) => ({
        response: response.data.authProviders,
    }));
}

/**
 * Fetches login authentication providers.
 *
 * @returns {Promise<Object, Error>} object with response property being an array of login auth providers
 */
export function fetchLoginAuthProviders() {
    return axios.get(`${authLoginProvidersUrl}`).then((response) => ({
        response: response.data.authProviders,
    }));
}

/**
 * Saves auth provider either by creating a new one (in case ID is missed) or by updating existing one by ID.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete
 */
export function saveAuthProvider(authProvider) {
    if (authProvider.active) {
        return authProvider.id;
    }
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
    return Promise.all(authProviderIds.map((id) => deleteAuthProvider(id)));
}

/*
 * Access Token Operations
 */

async function refreshAccessToken() {
    return axios
        .post(tokenRefreshUrl, null, doNotStallRequestConfig)
        .then(({ data: { token, expiry } }) => ({ token, info: { expiry } }));
}

const accessTokenManager = new AccessTokenManager({ refreshToken: refreshAccessToken });

export const getAccessToken = () => accessTokenManager.getToken();
export const storeAccessToken = (token) => accessTokenManager.setToken(token);

/**
 * Calls the server to check auth status, rejects with error if auth status isn't valid.
 * @returns {Promise<void>}
 */
export function getAuthStatus() {
    return axios.get('/v1/auth/status').then(({ data }) => {
        const { expires, refreshUrl, ...userAuthData } = data;
        // while it's a side effect, it's the best place to do it
        accessTokenManager.updateTokenInfo({ expiry: expires });
        return userAuthData;
    });
}

/**
 * Exchanges an external auth token for a Rox auth token.
 *
 * @param token the external auth token
 * @param type the type of authentication provider
 * @param state the state parameter
 * @returns {Promise} promise which is fulfilled when the request is complete
 */
export function exchangeAuthToken(token, type, state) {
    const data = {
        external_token: token,
        type,
        state,
    };
    return axios.post(`${authProvidersUrl}/exchangeToken`, data).then((response) => response.data);
}

/**
 * Terminates user's session with the backend and clears access token.
 */
export async function logout() {
    try {
        await axios.post(logoutUrl);
    } catch (e) {
        // regardless of the result proceed with token deletion
    }
    accessTokenManager.clearToken();
}

export const storeRequestedLocation = (location) => store.set(requestedLocationKey, location);
export const getAndClearRequestedLocation = () => {
    const location = store.get(requestedLocationKey);
    store.remove(requestedLocationKey);
    return location;
};

/**
 * Logs user in using the provided credentials for basic auth.
 * @returns {Promise} promise which is fulfilled when the request is complete or gets rejected with the error from the server.
 */
export function loginWithBasicAuth(username, password, authProvider) {
    const basicAuthPseudoToken = queryString.stringify({ username, password });
    return exchangeAuthToken(basicAuthPseudoToken, authProvider.type, authProvider.id).then(
        ({ token }) => {
            storeAccessToken(token);
            window.location = getAndClearRequestedLocation() || '/';
        }
    );
}

const BEARER_TOKEN_PREFIX = `Bearer `;

function setAuthHeader(config, token) {
    const {
        headers: { Authorization, ...notAuthHeaders },
    } = config;
    // make sure new config doesn't have unnecessary auth header
    const newConfig = {
        ...config,
        headers: {
            ...notAuthHeaders,
        },
    };
    if (token) newConfig.headers.Authorization = `${BEARER_TOKEN_PREFIX}${token}`;

    return newConfig;
}

function extractAccessTokenFromRequestConfig({ headers }) {
    if (
        !headers ||
        typeof headers.Authorization !== 'string' ||
        !headers.Authorization.startsWith(BEARER_TOKEN_PREFIX)
    ) {
        return null;
    }
    return headers.Authorization.substring(BEARER_TOKEN_PREFIX.length);
}

const parseAccessToken = (token) => {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
        atob(base64)
            .split('')
            .map((c) => {
                return `%${`00${c.charCodeAt(0).toString(16)}`.slice(-2)}`;
            })
            .join('')
    );
    return JSON.parse(jsonPayload);
};

export const getUserName = () => {
    const tokenInfo = parseAccessToken(getAccessToken());
    // in cypress tests we don't have an external_user field, but we do have a name field
    const { name, external_user: externalUser } = tokenInfo;
    if (name) return name;
    return externalUser.full_name || 'Admin';
};

function addAuthHeaderRequestInterceptor() {
    axios.interceptors.request.use(
        (config) => setAuthHeader(config, getAccessToken()),
        (error) => Promise.reject(error)
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

    addAuthHeaderRequestInterceptor();
    addTokenRefreshInterceptors(axios, accessTokenManager, {
        extractAccessToken: extractAccessTokenFromRequestConfig,
        handleAuthError: (error) => {
            const authError = new AuthHttpError(
                'Authentication Error',
                error.response.status,
                error
            );

            if (authError.isInvalidAuth()) {
                // clear token since it's not valid
                accessTokenManager.clearToken();
            }
            authHttpErrorHandler(authError);
        },
    });

    interceptorsAdded = true;
}
