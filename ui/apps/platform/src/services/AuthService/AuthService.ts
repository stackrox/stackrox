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
    requiredAttributes: AuthProviderRequiredAttribute[];
    traits?: Traits;
    claimMappings: Record<string, string> | [string, string][];
    lastUpdated: string;
};

export type AuthProviderInfo = {
    label: string;
    value: AuthProviderType;
};

export type AuthProviderRequiredAttribute = {
    attributeKey: string;
    attributeValue: string;
};

/**
 * Fetch authentication providers.
 */
export function fetchAuthProviders(): Promise<{ response: AuthProvider[] }> {
    return axios.get(`${authProvidersUrl}`).then((response) => {
        const authProviders = response?.data?.authProviders ?? [];
        return { response: authProviders.map((ap) => convertAuthProviderClaimMappingsToArray(ap)) };
    });
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

/**
 * Saves auth provider either by creating a new one (in case ID is missed) or by updating existing one by ID.
 */
export function saveAuthProvider(authProvider: AuthProvider): string | Promise<AuthProvider> {
    if (authProvider.active || getIsAuthProviderImmutable(authProvider)) {
        return authProvider.id;
    }

    return authProvider.id
        ? axios
              .put<AuthProvider>(
                  `${authProvidersUrl}/${authProvider.id}`,
                  convertAuthProviderClaimMappingsToObject(authProvider)
              )
              .then((response) => {
                  return convertAuthProviderClaimMappingsToArray(response.data);
              })
        : axios.post(authProvidersUrl, convertAuthProviderClaimMappingsToObject(authProvider));
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

function convertAuthProviderClaimMappingsToArray(provider: AuthProvider): AuthProvider {
    if (!provider.claimMappings) {
        return provider;
    }
    const mappingAsArray = Object.entries(provider.claimMappings).sort((a, b) =>
        a[0].localeCompare(b[0])
    );
    const newProvider = { ...provider };
    newProvider.claimMappings = mappingAsArray;
    return newProvider;
}

function convertAuthProviderClaimMappingsToObject(authProvider: AuthProvider): AuthProvider {
    if (!Array.isArray(authProvider.claimMappings)) {
        return authProvider;
    }
    const newProvider = { ...authProvider };
    newProvider.claimMappings = Object.fromEntries(authProvider.claimMappings);
    return newProvider;
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

export const dispatchResponseStarted = () => accessTokenManager.onDispatchResponseStarted();
export const dispatchResponseFinished = () => accessTokenManager.onDispatchResponseFinished();

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
    return exchangeAuthToken(basicAuthPseudoToken, authProvider.type, authProvider.id).then(() => {
        // window.location.href might be better, however
        // @ts-ignore 2322
        window.location = getAndClearRequestedLocation() || '/';
    });
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

    addTokenRefreshInterceptors(axios, accessTokenManager, {
        handleAuthError: (error) => {
            const authError = new AuthHttpError(
                'Authentication Error',
                error.response.status,
                error
            );
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
