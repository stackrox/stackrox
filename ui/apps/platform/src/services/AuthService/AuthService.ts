/* eslint-disable @typescript-eslint/ban-ts-comment */
import store from 'store';

import axios from 'services/instance';
import queryString from 'qs';

import { Role } from 'services/RolesService';

import { Empty } from 'services/types';
import AccessTokenManager from './AccessTokenManager';
import addTokenRefreshInterceptors, {
    doNotStallRequestConfig,
} from './addTokenRefreshInterceptors';
import { authProviderLabels } from '../../constants/accessControl';
import { Traits } from '../../types/traits.proto';
import { isUserResource } from '../../Containers/AccessControl/traits';

const authProvidersUrl = '/v1/authProviders';
const authLoginProvidersUrl = '/v1/login/authproviders';
const availableProviderTypesUrl = '/v1/availableAuthProviders';
const tokenRefreshUrl = '/sso/session/tokenrefresh';
const logoutUrl = '/sso/session/logout';

const requestedLocationKey = 'requested_location';

/**
 * Authentication HTTP Error that encapsulates HTTP errors related to user authentication and authorization.
 */
export class AuthHttpError extends Error {
    code: number;
    cause: Error; // eslint-disable-line @typescript-eslint/lines-between-class-members

    constructor(message: string, code: number, cause: Error) {
        super(message);
        this.name = 'AuthHttpError';
        this.code = code;
        /*
         * Although ES2022 adds `{ cause }` as optional argument to Error constructor
         * declare and assign cause property in subclass for backward compatibility.
         */
        this.cause = cause;
    }

    isAccessDenied = (): boolean => this.code === 403;

    isInvalidAuth = (): boolean => this.code === 401;
}

export type AuthProviderType = 'auth0' | 'oidc' | 'saml' | 'userpki' | 'iap' | 'openshift';

export type AuthProviderConfig = Record<
    string,
    string | number | undefined | boolean | Record<string, boolean>
>;

export type Group = {
    roleName: string;
    props: {
        authProviderId: string;
        key?: string;
        value?: string;
        id?: string;
        traits?: Traits | null;
    };
};

export type AuthProvider = {
    id: string;
    name: string;
    type: AuthProviderType;
    uiEndpoint?: string;
    enabled?: boolean;
    config: AuthProviderConfig;
    loginUrl?: string;
    extraUiEndpoints?: string[];
    active?: boolean;
    groups?: Group[];
    defaultRole?: string;
    requiredAttributes: AuthProviderRequiredAttributes[];
    traits?: Traits;
    claimMappings: Record<string, string>;
    lastUpdated: string;
};

export type AuthProviderInfo = {
    label: string;
    value: AuthProviderType;
};

export type AuthProviderRequiredAttributes = {
    attributeKey: string;
    attributeValue: string;
};

/**
 * Fetch authentication providers.
 */
export function fetchAuthProviders(): Promise<{ response: AuthProvider[] }> {
    return axios.get(`${authProvidersUrl}`).then((response) => ({
        response: response?.data?.authProviders ?? [],
    }));
}

export type AuthProviderLogin = {
    id: string;
    name: string;
    type: string;
    loginUrl: string;
};

/**
 * Fetch login authentication providers.
 */
export function fetchLoginAuthProviders(): Promise<{ response: AuthProviderLogin[] }> {
    return axios.get(`${authLoginProvidersUrl}`).then((response) => ({
        response: response?.data?.authProviders ?? [],
    }));
}

export function fetchAvailableProviderTypes(): Promise<{ response: AuthProviderInfo[] }> {
    return axios.get(`${availableProviderTypesUrl}`).then((response) => ({
        response:
            response?.data?.authProviderTypes?.map(({ type, suggestedAttributes }) => {
                return {
                    value: type,
                    ruleAttributes: suggestedAttributes,
                    label: authProviderLabels[type],
                };
            }) ?? [],
    }));
}

/*
 * Create entity and return object with id assigned by backend.
 */
export function createAuthProvider(authProvider: AuthProvider): Promise<AuthProvider> {
    return axios.post<AuthProvider>(authProvidersUrl, authProvider).then((response) => {
        return response.data;
    });
}

/*
 * Update entity and return object.
 */
export function updateAuthProvider(authProvider: AuthProvider): Promise<AuthProvider> {
    return axios
        .put<AuthProvider>(`${authProvidersUrl}/${authProvider.id}`, authProvider)
        .then((response) => {
            return response.data;
        });
}

/**
 * Saves auth provider either by creating a new one (in case ID is missed) or by updating existing one by ID.
 */
export function saveAuthProvider(authProvider: AuthProvider): string | Promise<AuthProvider> {
    if (authProvider.active || getIsAuthProviderImmutable(authProvider)) {
        return authProvider.id;
    }
    return authProvider.id
        ? axios.put(`${authProvidersUrl}/${authProvider.id}`, authProvider)
        : axios.post(authProvidersUrl, authProvider);
}

/**
 * Deletes auth provider by its ID.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete TODO verify return empty object
 */
export function deleteAuthProvider(authProviderId: string): Promise<Empty> {
    if (!authProviderId) {
        throw new Error('Auth provider ID must be defined');
    }
    return axios.delete(`${authProvidersUrl}/${authProviderId}`);
}

/**
 * Deletes auth providers by a list of IDs.
 *
 * @returns {Promise} promise which is fullfilled when the request is complete TODO return what?
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

// @ts-ignore 2322
const accessTokenManager = new AccessTokenManager({ refreshToken: refreshAccessToken });

export const getAccessToken = () => accessTokenManager.getToken();
export const storeAccessToken = (token) => accessTokenManager.setToken(token);

export type UserAttribute = {
    key: string;
    values: string[];
};

export type UserInfo = {
    username: string;
    friendlyName: string;
    permissions: { resourceToAccess: Record<string, string> };
    roles: Role[];
};

export type AuthStatus = {
    userId: string;
    // serviceId: string;
    expires: string; // ISO 8601 data string
    refreshUrl: string;
    authProvider: AuthProvider;
    userInfo: UserInfo;
    userAttributes: UserAttribute[];
};

export type UserAuthStatus = {
    userId: string;
    // serviceId: string;
    authProvider: AuthProvider;
    userInfo: UserInfo;
    userAttributes: UserAttribute[];
};

/**
 * Calls the server to check auth status, rejects with error if auth status isn't valid.
 * @returns {Promise<void>} TODO verify return UserAuthStatus instead of void
 */
export function getAuthStatus(): Promise<UserAuthStatus> {
    return axios.get<AuthStatus>('/v1/auth/status').then(({ data }) => {
        // disable because unused refreshUrl might be specified for rest spread idiom.
        /* eslint-disable @typescript-eslint/no-unused-vars */
        const { expires, refreshUrl, ...userAuthData } = data;
        /* eslint-enable @typescript-eslint/no-unused-vars */
        // while it's a side effect, it's the best place to do it
        // @ts-ignore 2345
        accessTokenManager.updateTokenInfo({ expiry: expires });
        return userAuthData;
    });
}

export type ExchangeTokenResponse = {
    token: string;
    clientState: string;
    test: boolean;
    user: AuthStatus;
};

/**
 * Exchanges an external auth token for a Rox auth token.
 */
export function exchangeAuthToken(
    token: string, // external auth token
    type: string, // type of authentication provider
    state: string
): Promise<ExchangeTokenResponse> {
    const data = {
        external_token: token,
        type,
        state,
    };
    return axios
        .post<ExchangeTokenResponse>(`${authProvidersUrl}/exchangeToken`, data)
        .then((response) => response.data);
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

export const storeRequestedLocation = (location: string): string =>
    store.set(requestedLocationKey, location) as string; // return location
export const getAndClearRequestedLocation = (): string => {
    const location = store.get(requestedLocationKey);
    store.remove(requestedLocationKey);
    return location as string;
};

/**
 * Logs user in using the provided credentials for basic auth.
 * @returns {Promise} promise which is fulfilled when the request is complete or gets rejected with the error from the server.
 */
export function loginWithBasicAuth(
    username: string,
    password: string,
    authProvider: AuthProvider
): Promise<void> {
    const basicAuthPseudoToken = queryString.stringify({ username, password });
    return exchangeAuthToken(basicAuthPseudoToken, authProvider.type, authProvider.id).then(
        ({ token }) => {
            storeAccessToken(token);
            // window.location.href might be better, however
            // @ts-ignore 2322
            window.location = getAndClearRequestedLocation() || '/';
        }
    );
}

const BEARER_TOKEN_PREFIX = `Bearer `;

function setAuthHeader(config, token: string) {
    // disable because unused Authorization might be specified for rest spread idiom.
    /* eslint-disable @typescript-eslint/no-unused-vars */
    const {
        headers: { Authorization, ...notAuthHeaders },
    } = config;
    /* eslint-enable @typescript-eslint/no-unused-vars */
    // make sure new config doesn't have unnecessary auth header
    const newConfig = {
        ...config,
        headers: {
            ...notAuthHeaders,
        },
    };
    if (token) {
        newConfig.headers.Authorization = `${BEARER_TOKEN_PREFIX}${token}`;
    }

    return newConfig as unknown;
}

function extractAccessTokenFromRequestConfig({ headers }) {
    if (
        !headers ||
        typeof headers.Authorization !== 'string' ||
        !headers.Authorization.startsWith(BEARER_TOKEN_PREFIX)
    ) {
        return null;
    }
    return headers.Authorization.substring(BEARER_TOKEN_PREFIX.length) as string;
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
    return JSON.parse(jsonPayload) as unknown;
};

export const getUserName = (): string => {
    const tokenInfo = parseAccessToken(getAccessToken());
    // in cypress tests we don't have an external_user field, but we do have a name field
    // @ts-ignore 2339
    const { name, external_user: externalUser } = tokenInfo;
    if (name) {
        return name as string;
    }
    return (externalUser.full_name as string) || 'Admin';
};

function addAuthHeaderRequestInterceptor() {
    axios.interceptors.request.use(
        // @ts-ignore 2345
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
export function addAuthInterceptors(authHttpErrorHandler): void {
    if (interceptorsAdded) {
        return;
    }

    addAuthHeaderRequestInterceptor();
    addTokenRefreshInterceptors(axios, accessTokenManager, {
        // @ts-ignore 2322
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

/**
 * Verifies whether the auth provider is immutable based on the traits property is set.
 * An auth provider is immutable if traits is undefined or is set to anything other than 'ALLOW_MUTATE'.
 *
 * @param {AuthProvider} authProvider auth provider to check.
 * @return {boolean} indicating whether the auth provider is immutable.
 */
export function getIsAuthProviderImmutable(authProvider: AuthProvider): boolean {
    return (
        ('traits' in authProvider &&
            authProvider.traits != null &&
            authProvider.traits?.mutabilityMode !== 'ALLOW_MUTATE') ||
        // Having both these conditions checked allows for seamless transition period
        // between using mutabilityMode and origin in ACSCS auth provider.
        !isUserResource(authProvider.traits)
    );
}
